package build

import (
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/schema"
)

type StepAddStream struct {
	Style         string                        // Style of input widget to use. Options are: "chooser"  and "inline".  Defaults to "chooser".
	Title         string                        // Title to use on the create modal. Defaults to "Add a Stream"
	Location      string                        // Options are: "top", "child", "outbox".  Defaults to "child".
	TemplateID    string                        // ID of the template to use.  If empty, then template roles are used.
	TemplateRoles []string                      // List of acceptable Template Roles that can be used to make a stream.  If empty, then all template for this container are valid.
	WithData      map[string]*template.Template // Map of values to preset in the new stream
}

/******************************************
 * GET Methods
 ******************************************/

// Get builds the HTML for this step - either a modal template selector, or the embedded edit form
func (step StepAddStream) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Style == "inline" {

		if err := step.getInline(builder, buffer); err != nil {
			return Halt().WithError(err)
		}
		return nil
	}

	// Fall through to displaying the default "CHOOSER" modal
	if err := step.getChooser(builder, buffer); err != nil {
		return Halt().WithError(err)
	}

	return Halt()
}

// modalAddStream builds an HTML dialog that lists all of the templates that the user can create
// tempalteIDs is a limiter on the list of valid templates.  If it is empty, then all valid templates are displayed.
func (step StepAddStream) getChooser(builder Builder, buffer io.Writer) error {
	// response *echo.Response, templateService *service.Template, iconProvider icon.Provider, title string, buffer io.Writer, url string, parentRole string, allowedTemplateIDs []string) {

	factory := builder.factory()
	response := builder.response()
	templateService := factory.Template()
	iconProvider := factory.Icons()
	parentRole := step.parentRole(builder)

	templates := templateService.ListByContainerLimited(parentRole, step.TemplateRoles)

	b := html.New()

	b.H2().InnerText(step.Title).Close()

	b.Table().Class("table margin-bottom")

	for _, template := range templates {
		b.TR().Role("link").Data("hx-post", builder.URL()+"?templateId="+template.Value)
		{
			b.TD().Class("text-3xl").Style("vertical-align:top").EndBracket()
			iconProvider.Write(template.Icon, b)
			// b.I(template.Icon, "text-3xl", "gray80").Close()
			b.Close()

			b.TD().Style("width:100%")
			b.Div().Class("bold").InnerText(template.Label).Close()
			b.Div().Class("gray60").InnerText(template.Description).Close()
			b.Close()
		}
		b.Close()
	}

	b.CloseAll()

	result := WrapModalWithCloseButton(response, b.String())

	// nolint:errcheck
	io.WriteString(buffer, result)

	return nil
}

// getInline builds the HTML for an embedded form
func (step StepAddStream) getInline(builder Builder, buffer io.Writer) error {

	const location = "build.StepAddStream.getInline"

	// Get prerequisites
	factory := builder.factory()
	templateService := factory.Template()
	containedByRole := step.parentRole(builder)

	// Find the "selected" template
	optionTemplates, newTemplate, err := step.getBestTemplate(templateService, containedByRole, builder.QueryParam("templateId"))

	if err != nil {
		return derp.Wrap(err, location, "Error getting best template")
	}

	iconService := builder.factory().Icons()

	path := builder.request().URL.Path
	path = replaceActionID(path, builder.actionID())

	// Build the HTML for the "embed" widget
	b := html.New()
	b.Div().Data("hx-target", "this").Data("hx-swap", "outerHTML").Data("hx-push-url", "false").EndBracket()

	if len(optionTemplates) > 1 {
		b.Div()
		for _, optionTemplate := range optionTemplates {

			b.A("").Data("hx-get", path+"?templateId="+optionTemplate.Value).Class("align-center", "inline-block", "margin-right-md").EndBracket()

			b.Div().Class("text-lg", "margin-vertical-none").EndBracket()
			if newTemplate.TemplateID == optionTemplate.Value {
				iconService.Write(optionTemplate.Icon+"-fill", b)
			} else {
				iconService.Write(optionTemplate.Icon, b)
			}
			b.Close() // DIV

			b.Div().Class("margin-vertical-none", "text-sm").InnerText(optionTemplate.Label).Close()

			b.Close() // A

			b.WriteString("&nbsp;")
		}
		b.Close() // DIV
	}

	// If there is a child builder, then build it here

	// Create a new child stream
	streamService := factory.Stream()
	child := streamService.New()

	if user, err := builder.getUser(); err == nil {
		child.SetAttributedTo(user.PersonLink())
	}

	// Apply custom stream data from the "with-data" map
	if err := step.setStreamData(builder, &child); err != nil {
		return derp.Wrap(err, location, "Error setting stream data")
	}

	// Create a new child builder
	childBuilder, err := NewStream(factory, builder.request(), builder.response(), newTemplate, &child, "create")
	childBuilder.setArguments(builder.getArguments())

	if err != nil {
		return derp.Wrap(err, location, "Error creating new child stream builder")
	}

	widgetHTML, err := childBuilder.Render()
	if err != nil {
		return derp.Wrap(err, location, "Error building new child stream")
	}

	b.WriteString(string(widgetHTML))

	// Close the container
	b.Close()

	// Write the whole widget back to the outpub buffer
	// nolint:errcheck
	buffer.Write(b.Bytes())
	return nil
}

/******************************************
 * POST Methods
 ******************************************/

func (step StepAddStream) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepAddStream.Post"

	// Collect prerequisites
	factory := builder.factory()
	templateService := factory.Template()
	containedByRole := step.parentRole(builder)

	// Identify the Template to used for the new Stream
	_, template, err := step.getBestTemplate(templateService, containedByRole, builder.QueryParam("templateId"))

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Invalid Template. Check template roles and 'containedBy' values."))
	}

	// Create the new Stream
	streamService := factory.Stream()
	newStream := streamService.New()

	// Assign the current user as the author (with silent failure, but why would it do that?)
	if user, err := builder.getUser(); err == nil {
		newStream.SetAttributedTo(user.PersonLink())
	}

	// Validate and set the location for the new Stream
	if err := step.setLocation(builder, &template, &newStream); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error getting location for new stream"))
	}

	// Apply custom stream data from the "with-data" map
	if err := step.setStreamData(builder, &newStream); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error setting stream data"))
	}

	// Create a builder for the new Stream
	newBuilder, err := NewStream(factory, builder.request(), builder.response(), template, &newStream, "create")
	newBuilder.setArguments(builder.getArguments())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating builder", newStream))
	}

	// Run the "create" action for the new stream's template, if possible
	result := Pipeline(newBuilder.action().Steps).Post(factory, newBuilder, buffer)
	result.Error = derp.Wrap(result.Error, location, "Unable to execute 'create' action on stream")

	// For "inline" styles, use the result from the child's "create" action
	// to determine what happens next.
	if step.Style == "inline" {
		return UseResult(result).AsFullPage()
	}

	// For "chooser" style, close window and go to the "edit"
	// route of the new Stream
	return UseResult(result).WithEvent("closeModal", "")
}

// setLocation returns the ParentIDs to use in for the new stream.
// It returns an error if the template cannot be placed in the pre-determined location.
func (step StepAddStream) setLocation(builder Builder, template *model.Template, newStream *model.Stream) error {

	const location = "build.StepAddStream.setLocation"

	streamService := builder.factory().Stream()

	switch step.Location {

	// Special case for streams in User's Outbox
	case "outbox":

		userID := builder.AuthenticatedID()
		if err := streamService.SetLocationOutbox(template, newStream, userID); err != nil {
			return derp.Wrap(err, location, "Error setting location for outbox")
		}
		return nil

	// Special case for "Top-Level" Navigation
	case "top":

		if err := streamService.SetLocationTop(template, newStream); err != nil {
			return derp.Wrap(err, location, "Error setting location for top")
		}
		return nil

	// Default to "Child" streams
	default:

		// Guarantee that the current builder is a Stream builder
		streamBuilder, ok := builder.(Stream)

		if !ok {
			return derp.NewForbiddenError(location, "Cannot add child stream to non-stream builder")
		}

		parent := streamBuilder._stream
		if err := streamService.SetLocationChild(template, newStream, parent); err != nil {
			return derp.Wrap(err, step.Location, "Error setting location for child")
		}
		return nil
	}
}

// setStreamData applies the "with-data" map to the newly created stream
func (step StepAddStream) setStreamData(builder Builder, stream *model.Stream) error {

	if len(step.WithData) == 0 {
		return nil
	}

	s := schema.New(model.StreamSchema())

	for key, valueTemplate := range step.WithData {
		value := executeTemplate(valueTemplate, builder)
		if err := s.Set(stream, key, value); err != nil {
			return derp.Wrap(err, "build.StepAddStream.setStreamData", "Error setting stream data", key, value)
		}
	}

	return nil
}

// getBastTemplate applies several rules to determine which template can be used for a new Stream.
// It returns a slice of eligible Templates (as form.LookupCodes), and the selected Template.
func (step StepAddStream) getBestTemplate(templateService *service.Template, containedByRole string, selectedTemplateID string) ([]form.LookupCode, model.Template, error) {

	const location = "build.StepAddStream.getBestTemplate"

	// Query all eligible Templates
	eligible := templateService.ListByContainerLimited(containedByRole, step.TemplateRoles)

	// If NO Templates are eligible, then return empty string
	if len(eligible) == 0 {
		return []form.LookupCode{}, model.Template{}, derp.NewInternalError(location, "No eligible Templates provided")
	}

	// If the Step already has a Template defined, then this overrides the passed-in value
	if step.TemplateID != "" {
		for index, eligibleTemplate := range eligible {
			if eligibleTemplate.Value == step.TemplateID {
				return step.getBestTemplate_result(templateService, eligible[index:index+1], step.TemplateID)
			}
		}
		return []form.LookupCode{}, model.Template{}, derp.NewInternalError(location, "Template '"+step.TemplateID+"' (defined in this Step) cannot be placed within '"+containedByRole+"'")
	}

	// Search eligible templates for the selected TemplateID, returning when found
	for _, template := range eligible {
		if template.Value == selectedTemplateID {
			return step.getBestTemplate_result(templateService, eligible, selectedTemplateID)
		}
	}

	// None found. Use the first "eligible" template"
	return step.getBestTemplate_result(templateService, eligible, eligible[0].Value)
}

// getBestTemplate_result finishes the job of getBestTemplate by loading the selected Template from the memory-cache
func (step StepAddStream) getBestTemplate_result(templateService *service.Template, eligible []form.LookupCode, templateID string) ([]form.LookupCode, model.Template, error) {

	const location = "build.StepAddStream.getBestTemplate_result"

	template, err := templateService.Load(templateID)

	if err != nil {
		return []form.LookupCode{}, model.Template{}, derp.Wrap(err, location, "Error loading Template selected by User", eligible, templateID)
	}

	return eligible, template, nil
}

func (step StepAddStream) parentRole(builder Builder) string {

	if step.Location == "child" {
		return builder.templateRole()
	}

	return step.Location
}
