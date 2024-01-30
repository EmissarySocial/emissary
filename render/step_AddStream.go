package render

import (
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// Get renders the HTML for this step - either a modal template selector, or the embedded edit form
func (step StepAddStream) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	if step.Style == "inline" {

		if err := step.getInline(renderer, buffer); err != nil {
			return Halt().WithError(err)
		}
		return nil
	}

	// Fall through to displaying the default "CHOOSER" modal
	if err := step.getChooser(renderer, buffer); err != nil {
		return Halt().WithError(err)
	}

	return Halt()
}

// modalAddStream renders an HTML dialog that lists all of the templates that the user can create
// tempalteIDs is a limiter on the list of valid templates.  If it is empty, then all valid templates are displayed.
func (step StepAddStream) getChooser(renderer Renderer, buffer io.Writer) error {
	// response *echo.Response, templateService *service.Template, iconProvider icon.Provider, title string, buffer io.Writer, url string, parentRole string, allowedTemplateIDs []string) {

	factory := renderer.factory()
	response := renderer.response()
	templateService := factory.Template()
	iconProvider := factory.Icons()
	parentRole := step.parentRole(renderer)

	templates := templateService.ListByContainerLimited(parentRole, step.TemplateRoles)

	b := html.New()

	b.H2().InnerText(step.Title).Close()

	b.Table().Class("table margin-bottom")

	for _, template := range templates {
		b.TR().Role("link").Data("hx-post", renderer.URL()+"?templateId="+template.Value)
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

// getInline renders the HTML for an embedded form
func (step StepAddStream) getInline(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAddStream.getInline"

	// Get prerequisites
	factory := renderer.factory()
	templateService := factory.Template()
	parentRole := step.parentRole(renderer)

	// Query all eligible templates
	templates := templateService.ListByContainerLimited(parentRole, step.TemplateRoles)

	if len(templates) == 0 {
		return derp.NewBadRequestError(location, "No child templates available for this Role", renderer.templateRole())
	}

	// Find the "selected" template
	templateID := step.getBestTemplate(templates, renderer.QueryParam("templateId"))

	iconService := renderer.factory().Icons()

	path := renderer.request().URL.Path
	path = replaceActionID(path, renderer.ActionID())

	// Build the HTML for the "embed" widget
	b := html.New()
	b.Div().Data("hx-target", "this").Data("hx-swap", "outerHTML").Data("hx-push-url", "false").EndBracket()

	if len(templates) > 1 {
		b.Div()
		for _, template := range templates {

			b.A("").Data("hx-get", path+"?templateId="+template.Value).Class("align-center", "inline-block", "margin-right-md").EndBracket()

			b.Div().Class("text-lg", "margin-vertical-none").EndBracket()
			if templateID == template.Value {
				iconService.Write(template.Icon+"-fill", b)
			} else {
				iconService.Write(template.Icon, b)
			}
			b.Close() // DIV

			b.Div().Class("margin-vertical-none", "text-sm").InnerText(template.Label).Close()

			b.Close() // A

			b.WriteString("&nbsp;")
		}
		b.Close() // DIV
	}

	// If there is a child renderer, then render it here

	// Create a new child stream
	streamService := factory.Stream()
	child, template, err := streamService.New(renderer.NavigationID(), renderer.objectID(), templateID)

	if err != nil {
		return derp.Wrap(err, location, "Error creating new child stream")
	}

	// Create a new child renderer
	childRenderer, err := NewStream(factory, renderer.request(), renderer.response(), template, &child, "create")
	childRenderer.setArguments(renderer.getArguments())

	if err != nil {
		return derp.Wrap(err, location, "Error creating new child stream renderer")
	}

	widgetHTML, err := childRenderer.Render()
	if err != nil {
		return derp.Wrap(err, location, "Error rendering new child stream")
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

func (step StepAddStream) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {

	const location = "render.StepAddStream.Post"

	// Collect prerequisites
	factory := renderer.factory()
	templateService := factory.Template()
	parentRole := step.parentRole(renderer)

	// Query all eligible Templates
	templates := templateService.ListByContainerLimited(parentRole, step.TemplateRoles)

	if len(templates) == 0 {
		return Halt().WithError(derp.NewBadRequestError(location, "No child templates available for this Role", renderer.templateRole()))
	}

	// Identify the Template to used for the new Stream
	templateID := step.getBestTemplate(templates, renderer.QueryParam("templateId"))

	if templateID == "" {
		return Halt().WithError(derp.NewBadRequestError(location, "Invalid Template. Check template roles and 'containedBy' values."))
	}

	// Create the new Stream
	streamService := factory.Stream()
	newStream, newTemplate, err := streamService.New("", primitive.NilObjectID, templateID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating new stream", templateID))
	}

	// Validate and set the location for the new Stream
	if err := step.setLocation(renderer, &newTemplate, &newStream); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error getting location for new stream"))
	}

	// Apply custom stream data from the "with-data" map
	if err := step.setStreamData(renderer, &newStream); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error setting stream data"))
	}

	// Assign the current user as the author (with silent failure)
	if user, err := renderer.getUser(); err == nil {
		newStream.SetAttributedTo(user.PersonLink())
	}

	// Create a renderer for the new Stream
	newRenderer, err := NewStream(factory, renderer.request(), renderer.response(), newTemplate, &newStream, "create")
	newRenderer.setArguments(renderer.getArguments())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating renderer", newStream))
	}

	// Run the "create" action for the new stream's template, if possible
	result := Pipeline(newRenderer.Action().Steps).Post(factory, &newRenderer, buffer)
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
func (step StepAddStream) setLocation(renderer Renderer, template *model.Template, newStream *model.Stream) error {

	const location = "render.StepAddStream.setLocation"

	switch step.Location {

	// Special case for streams in User's Outbox
	case "outbox":
		userID := renderer.AuthenticatedID()

		if userID.IsZero() {
			return derp.NewUnauthorizedError(location, "Cannot add to outbox because user is not authenticated")
		}

		if !template.CanBeContainedBy("outbox") {
			return derp.NewBadRequestError(location, "Template cannot be placed in the outbox", template.TemplateID)
		}

		newStream.NavigationID = "profile"
		newStream.ParentID = userID
		newStream.ParentIDs = []primitive.ObjectID{}

		return nil

	// Special case for "Top-Level" Navigation
	case "top":

		if !template.CanBeContainedBy("top") {
			return derp.NewBadRequestError(location, "Template cannot be placed in the top navigation", template.TemplateID)
		}

		newStream.NavigationID = newStream.StreamID.Hex()
		newStream.ParentID = primitive.NilObjectID
		newStream.ParentIDs = []primitive.ObjectID{}

		return nil

	// Default to "Child" streams
	default:

		// Guarantee that the current renderer is a Stream renderer
		streamRenderer, ok := renderer.(*Stream)

		if !ok {
			return derp.NewForbiddenError(location, "Cannot add child stream to non-stream renderer")
		}

		// Look up the TemplateRole for the current Stream
		parent := streamRenderer._stream
		parentTemplate, err := renderer.factory().Template().Load(parent.TemplateID)

		if err != nil {
			return derp.Wrap(err, location, "Error loading parent template")
		}

		// Guarantee that the selected Template can be contained by the parent Template
		if !template.CanBeContainedBy(parentTemplate.TemplateRole) {
			return derp.NewBadRequestError(location, "Child cannot be placed in this parent template", parentTemplate.TemplateRole, template.TemplateID)
		}

		// Set values and exit
		newStream.NavigationID = parent.NavigationID
		newStream.ParentID = parent.StreamID
		newStream.ParentIDs = append(parent.ParentIDs, parent.StreamID)

		return nil
	}
}

func (step StepAddStream) setStreamData(renderer Renderer, stream *model.Stream) error {

	if len(step.WithData) == 0 {
		return nil
	}

	s := schema.New(model.StreamSchema())

	for key, valueTemplate := range step.WithData {
		value := executeTemplate(valueTemplate, renderer)
		if err := s.Set(stream, key, value); err != nil {
			return derp.Wrap(err, "render.StepAddStream.setStreamData", "Error setting stream data", key, value)
		}
	}

	return nil
}

// getBastTemplate applies several rules to determine which template to use for the new stream
func (step StepAddStream) getBestTemplate(eligible []form.LookupCode, templateID string) string {

	// If NO Templates are eligible, then return empty string
	if len(eligible) == 0 {
		return ""
	}

	// If the Step already has a Template defined, then this overrides the passed-in value
	if step.TemplateID != "" {
		templateID = step.TemplateID
	}

	// Search eligible templates for the selected TemplateID, returning when found
	for _, template := range eligible {
		if template.Value == templateID {
			return templateID
		}
	}

	// If not found, then return the first eligible template
	return eligible[0].Value
}

func (step StepAddStream) parentRole(renderer Renderer) string {

	if step.Location == "child" {
		return renderer.templateRole()
	}

	return step.Location
}
