package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/benpate/html"
)

// StepNewChild is an action that can add new sub-streams to the domain.
type StepNewChild struct {
	templateService *service.Template
	streamService   *service.Stream
	templateIDs     []string
	childState      string
	withChild       []datatype.Map
}

// NewStepNewChild returns a fully initialized StepNewChild record
func NewStepNewChild(templateService *service.Template, streamService *service.Stream, stepInfo datatype.Map) StepNewChild {
	return StepNewChild{
		templateService: templateService,
		streamService:   streamService,
		templateIDs:     stepInfo.GetSliceOfString("template"),
		childState:      stepInfo.GetString("childState"),
		withChild:       stepInfo.GetSliceOfMap("withChild"),
	}
}

func (step StepNewChild) Get(buffer io.Writer, renderer Renderer) error {
	streamRenderer := renderer.(*Stream)
	modalNewChild(step.templateService, buffer, streamRenderer.URL(), streamRenderer.TemplateID(), step.templateIDs)
	return nil
}

func (step StepNewChild) Post(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(*Stream)
	templateID := streamRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.templateIDs) > 0 {

		if templateID == "" {
			templateID = step.templateIDs[0]
		} else if !compare.Contains(step.templateIDs, templateID) {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepNewChild.Post", "Cannot create new template of this kind", templateID)
		}
	}

	// Create new child stream
	child, template, err := step.streamService.NewChild(streamRenderer.stream, templateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating new child stream", templateID)
	}

	// Set Default Values

	child.StateID = step.childState
	childStream, err := NewStream(streamRenderer.factory, streamRenderer.context(), template, &child, "view")

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating renderer", child)
	}

	if err := childStream.setAuthor(); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error retrieving author inforation", child)
	}

	// If there is an "init" step for the child's template, then execute it now
	if action, ok := template.Action("init"); ok {
		if err := DoPipeline(streamRenderer.factory, &childStream, buffer, action.Steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute 'init' action on child")
		}
	}

	if child.IsNew() {
		if err := step.streamService.Save(&child, "Created"); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error saving child stream to database")
		}
	}

	if err := DoPipeline(streamRenderer.factory, &childStream, buffer, step.withChild, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute action steps on child")
	}

	return nil
}

// modalNewChild renders an HTML dialog that lists all of the templates that the user can create
// tempalteIDs is a limiter on the list of valid templates.  If it is empty, then all valid templates are displayed.
func modalNewChild(templateService *service.Template, buffer io.Writer, url string, parentTemplateID string, allowedTemplateIDs []string) {

	templates := templateService.ListByContainerLimited(parentTemplateID, allowedTemplateIDs)

	b := html.New()

	b.Div().ID("modal").Data("hx-swap", "none").Data("hx-push-url", "false")
	b.Div().Class("modal-underlay").Close()
	b.Div().Class("modal-content")
	b.H2().InnerHTML("+ Add a Stream").Close()
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

	b.Close()
	b.Div()
	b.Button().
		Script("on click trigger closeModal").
		InnerHTML("Cancel")

	b.CloseAll()

	buffer.Write([]byte(b.String()))
}
