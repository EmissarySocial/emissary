package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/tools/val"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StepAddStream struct {
	Title         string      // Title to use on the create modal. Defaults to "Add a Stream"
	Location      string      // Options are: "top", "child", "outbox".  Defaults to "child".
	Templates     []string    // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
	AsEmbed       bool        // If TRUE, then use embed the "create" action of the selected template into the current page.
	AsReply       bool        // If TRUE, then the new stream will be created as a reply to the current model object (only works with DocumentLinkers: Streams and Messages).
	WithNewStream []step.Step // List of steps to take on the newly created child record on POST.
}

// Get renders the HTML for this step - either a modal template selector, or the embedded edit form
func (step StepAddStream) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	// If this is an "Embedded" form, then call the appropriate fork.
	if step.AsEmbed {
		if err := step.getEmbed(renderer, buffer); err != nil {
			return ExitError(err)
		}
		return ExitFullPage()
	}

	// Fall through to displaying the default modal
	if err := step.getModal(renderer, buffer); err != nil {
		return ExitError(err)
	}

	return ExitFullPage()
}

func (step StepAddStream) Post(renderer Renderer, buffer io.Writer) ExitCondition {

	const location = "render.StepAddStream.Post"

	// Collect prerequisites
	factory := renderer.factory()
	context := renderer.context()
	templateID := context.QueryParam("templateId")

	if len(step.Templates) > 0 {
		templateID = val.Enum(templateID, step.Templates...)
	}

	// Try to load the template for the new stream
	newTemplate, err := factory.Template().Load(templateID)

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Template not found", templateID))
	}

	// Create the new child stream
	streamService := factory.Stream()
	newStream, _, err := streamService.New("", primitive.NilObjectID, templateID)

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Error creating new stream", templateID))
	}

	// Validate and set the location for the new stream
	if err := step.setLocation(renderer, &newTemplate, &newStream); err != nil {
		return ExitError(derp.Wrap(err, location, "Error getting location for new stream"))
	}

	// If this is a reply, then try to get a DocumentLink for the object we're replying to.
	if step.AsReply {
		if documentLinker, ok := renderer.object().(DocumentLinker); ok {
			newStream.InReplyTo = documentLinker.DocumentLink().URL
		} else {
			return ExitError(derp.NewInternalError(location, "Replies can only be made to Stream and Message (DocumentLinker) objects."))
		}
	}

	// Create a renderer for the new Stream
	newRenderer, err := NewStream(factory, context, newTemplate, &newStream, "view")

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Error creating renderer", newStream))
	}

	// Assign the current user as the author (with silent failure)
	if user, err := newRenderer.getUser(); err == nil {
		newRenderer.stream.SetAttributedTo(user.PersonLink())
	}

	// If there is an "init" step for the stream's template, then execute it now
	if action, ok := newTemplate.Action("init"); ok {
		status := Pipeline(action.Steps).Post(factory, &newRenderer, buffer)
		status.Error = derp.Wrap(status.Error, location, "Unable to execute 'init' action on stream")

		if status.Halt {
			return ExitWithStatus(status)
		}
	}

	// If this is an "embed" action, then also call the "create" action on the new stream
	if step.AsEmbed {
		if action, ok := newTemplate.Action("create"); ok {
			status := Pipeline(action.Steps).Post(factory, &newRenderer, buffer)
			status.Error = derp.Wrap(status.Error, location, "Unable to execute 'create' action on stream")

			if status.Halt {
				return ExitWithStatus(status)
			}
		}
	}

	// Execute additional "with-stream" steps
	status := Pipeline(step.WithNewStream).Post(factory, &newRenderer, buffer)
	status.Error = derp.Wrap(status.Error, location, "Unable to execute action steps on stream")

	return ExitWithStatus(status).AsFullPage()
}

// getEmbed renders the HTML for an embedded form
func (step StepAddStream) getEmbed(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAddStream.Get"

	// Get prerequisites
	factory := renderer.factory()
	context := renderer.context()
	templateService := factory.Template()
	parentRole := step.parentRole(renderer)

	// Query all eligible templates
	templates := templateService.ListByContainerLimited(parentRole, step.Templates)

	if len(templates) == 0 {
		return derp.NewBadRequestError(location, "No child templates available for this Role", renderer.templateRole())
	}

	// Find the "selected" template
	selectedTemplateID := step.getBestTemplate(templates, context.QueryParam("templateId"))

	iconService := renderer.factory().Icons()

	path := renderer.context().Request().URL.Path
	path = replaceActionID(path, renderer.ActionID())

	// Build the HTML for the "embed" widget
	b := html.New()
	b.Div().Data("hx-target", "this").Data("hx-swap", "outerHTML").EndBracket()

	if len(templates) > 1 {
		b.Div()
		for _, template := range templates {

			b.A("").Data("hx-get", path+"?templateId="+template.Value).Class("align-center", "inline-block", "space-right").EndBracket()

			b.Div().Class("text-lg", "vertical-space-none").EndBracket()
			if selectedTemplateID == template.Value {
				iconService.Write(template.Icon+"-fill", b)
			} else {
				iconService.Write(template.Icon, b)
			}
			b.Close() // DIV

			b.Div().Class("vertical-space-none", "text-sm").InnerText(template.Label).Close()

			b.Close() // A

			b.WriteString("&nbsp;")
		}
		b.Close() // DIV
	}

	// If there is a child renderer, then render it here

	// Create a new child stream
	streamService := factory.Stream()
	child, template, err := streamService.New(renderer.NavigationID(), renderer.objectID(), selectedTemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Error creating new child stream")
	}

	// Create a new child renderer
	childRenderer, err := NewStream(factory, context, template, &child, "create")

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

	// nolint:errcheck // Write the whole widget back to the outpub buffer
	buffer.Write(b.Bytes())
	return nil
}

// modalAddStream renders an HTML dialog that lists all of the templates that the user can create
// tempalteIDs is a limiter on the list of valid templates.  If it is empty, then all valid templates are displayed.
func (step StepAddStream) getModal(renderer Renderer, buffer io.Writer) error {
	// response *echo.Response, templateService *service.Template, iconProvider icon.Provider, title string, buffer io.Writer, url string, parentRole string, allowedTemplateIDs []string) {

	factory := renderer.factory()
	response := renderer.context().Response()
	templateService := factory.Template()
	iconProvider := factory.Icons()
	parentRole := step.parentRole(renderer)

	templates := templateService.ListByContainerLimited(parentRole, step.Templates)

	b := html.New()

	b.H2().InnerText(step.Title).Close()

	b.Table().Class("table space-below")

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

		streamRenderer, ok := renderer.(*Stream)

		if !ok {
			return derp.NewForbiddenError(location, "Cannot add child stream to non-stream renderer")
		}

		templateService := renderer.factory().Template()

		parent := streamRenderer.stream

		parentTemplate, err := templateService.Load(parent.TemplateID)

		if err != nil {
			return derp.Wrap(err, location, "Error loading parent template")
		}

		if !template.CanBeContainedBy(parentTemplate.TemplateRole) {
			return derp.NewBadRequestError(location, "Child cannot be placed in this parent template", parentTemplate.TemplateRole, template.TemplateID)
		}

		newStream.NavigationID = parent.NavigationID
		newStream.ParentID = parent.StreamID
		newStream.ParentIDs = append(parent.ParentIDs, parent.StreamID)

		return nil
	}
}

func (step StepAddStream) getBestTemplate(templates []form.LookupCode, templateID string) string {

	if len(templates) == 0 {
		return ""
	}

	for _, template := range templates {
		if template.Value == templateID {
			return templateID
		}
	}

	return templates[0].Value
}

func (step StepAddStream) parentRole(renderer Renderer) string {

	if step.Location == "child" {
		return renderer.templateRole()
	}

	return step.Location

}
