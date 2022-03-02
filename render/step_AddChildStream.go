package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/service"
)

// StepAddChildStream is an action that can add new sub-streams to the domain.
type StepAddChildStream struct {
	templateIDs []string       // List of acceptable templates that can be used to make a stream.  If empty, then all templates are valid.
	view        string         // If present, use this HTML template as a custom "create" page.  If missing, a default modal pop-up is used.
	withChild   []datatype.Map // List of steps to take on the newly created child record on POST.

	templateService *service.Template
	streamService   *service.Stream
}

// NewStepAddChildStream returns a fully initialized StepAddChildStream record
func NewStepAddChildStream(templateService *service.Template, streamService *service.Stream, stepInfo datatype.Map) StepAddChildStream {
	return StepAddChildStream{
		templateService: templateService,
		streamService:   streamService,
		view:            stepInfo.GetString("view"),
		templateIDs:     stepInfo.GetSliceOfString("template"),
		withChild:       stepInfo.GetSliceOfMap("with-child"),
	}
}

func (step StepAddChildStream) Get(buffer io.Writer, renderer Renderer) error {

	// This can only be used on a Stream Renderer
	streamRenderer := renderer.(*Stream)

	// If a view has been specified, then use it to render a "create" page
	if step.view != "" {

		if err := renderer.executeTemplate(buffer, step.view, renderer); err != nil {
			return derp.Wrap(err, "whisper.render.StepViewHTML.Get", "Error executing template")
		}

		return nil
	}

	// Fall through to displaying the default modal
	modalAddStream(renderer.context().Response(), step.templateService, buffer, streamRenderer.URL(), streamRenderer.TemplateID(), step.templateIDs)

	return nil
}

func (step StepAddChildStream) Post(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(*Stream)
	templateID := streamRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.templateIDs) > 0 {

		if templateID == "" {
			templateID = step.templateIDs[0]
		} else if !compare.Contains(step.templateIDs, templateID) {
			return derp.New(derp.CodeBadRequestError, "render.StepAddChildStream.Post", "Cannot create new template of this kind", templateID)
		}
	}

	// Create new child stream
	child, template, err := step.streamService.NewChild(streamRenderer.stream, templateID)

	if err != nil {
		return derp.Wrap(err, "render.StepAddChildStream.Post", "Error creating new child stream", templateID)
	}

	// Create child stream
	action := template.Action("view")
	childStream, err := NewStream(streamRenderer.factory(), streamRenderer.context(), template, action, &child)

	if err != nil {
		return derp.Wrap(err, "render.StepAddChildStream.Post", "Error creating renderer", child)
	}

	// Assign the current user as the author
	if err := childStream.setAuthor(); err != nil {
		return derp.Wrap(err, "render.StepAddChildStream.Post", "Error retrieving author inforation", child)
	}

	// If there is an "init" step for the child's template, then execute it now
	if action := template.Action("init"); action != nil {
		if err := DoPipeline(&childStream, buffer, action.Steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "render.StepAddChildStream.Post", "Unable to execute 'init' action on child")
		}
	}

	// If the child was not saved by the "init" steps, then save it now
	if child.IsNew() {
		if err := step.streamService.Save(&child, "Created"); err != nil {
			return derp.Wrap(err, "render.StepAddChildStream.Post", "Error saving child stream to database")
		}
	}

	// Execute additional "with-child" steps
	if len(step.withChild) > 0 {
		if err := DoPipeline(&childStream, buffer, step.withChild, ActionMethodPost); err != nil {
			return derp.Wrap(err, "render.StepAddChildStream.Post", "Unable to execute action steps on child")
		}
	}

	return nil
}

// modalAddStream renders an HTML dialog that lists all of the templates that the user can create
// tempalteIDs is a limiter on the list of valid templates.  If it is empty, then all valid templates are displayed.
func modalAddStream(response *echo.Response, templateService *service.Template, buffer io.Writer, url string, parentTemplateID string, allowedTemplateIDs []string) {

	templates := templateService.ListByContainerLimited(parentTemplateID, allowedTemplateIDs)

	b := html.New()

	b.H1().InnerHTML("+ Add a Stream").Close()
	b.Table().Class("table space-below")

	for _, template := range templates {
		b.TR().Data("hx-post", url+"?templateId="+template.Value)
		{
			b.TD()
			b.I(template.Icon + " fa-3x gray").Close()
			b.Close()

			b.TD().Style("width:100%")
			b.Div().Class("big", "bold").InnerHTML(template.Label).Close()
			b.Div().Class("small", "gray").InnerHTML(template.Description).Close()
			b.Close()
		}
		b.Close()
	}

	b.CloseAll()

	result := WrapModalWithCloseButton(response, b.String())

	io.WriteString(buffer, result)
}
