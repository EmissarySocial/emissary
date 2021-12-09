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

func (step StepNewChild) Get(buffer io.Writer, renderer *Stream) error {
	modalNewChild(step.templateService, buffer, renderer, step.templateIDs)
	return nil
}

func (step StepNewChild) Post(buffer io.Writer, renderer *Stream) error {

	templateID := renderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.templateIDs) > 0 {

		if templateID == "" {
			templateID = step.templateIDs[0]
		} else if !compare.Contains(step.templateIDs, templateID) {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepNewChild.Post", "Cannot create new template of this kind", templateID)
		}
	}

	// Create new child stream
	child, template, err := step.streamService.NewChild(renderer.stream, templateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating new child stream", templateID)
	}

	// Set Default Values
	authorization := getAuthorization(renderer.ctx)

	child.StateID = step.childState
	child.AuthorID = authorization.UserID
	childStream, err := renderer.newStream(&child, "edit")

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating renderer", child)
	}

	// If there is an "init" step for the child's template, then execute it now
	if action, ok := template.Action("init"); ok {
		if err := DoPipeline(&childStream, buffer, action.Steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute 'init' action on child")
		}
	}

	if child.IsNew() {
		if err := step.streamService.Save(&child, "Created"); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error saving child stream to database")
		}
	}

	if err := DoPipeline(&childStream, buffer, step.withChild, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute action steps on child")
	}

	return nil
}

// modalNewChild renders an HTML dialog that lists all of the templates that the user can create
// tempalteIDs is a limiter on the list of valid templates.  If it is empty, then all valid templates are displayed.
func modalNewChild(templateService *service.Template, buffer io.Writer, renderer *Stream, templateIDs []string) {

	templates := templateService.ListByContainerLimited(renderer.TemplateID(), templateIDs)

	b := html.New()

	b.Div().ID("modal")
	b.Div().Class("modal-underlay").Close()
	b.Div().Class("modal-content")
	b.H2().InnerHTML("+ Add a Stream").Close()
	b.Table().Class("table space-below")

	for _, template := range templates {
		b.TR().Data("hx-post", renderer.URL()+"?templateId="+template.Value)
		{
			b.TD()
			b.I(template.Icon).Close()
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
