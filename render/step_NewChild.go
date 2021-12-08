package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

// StepNewChild is an action that can add new sub-streams to the domain.
type StepNewChild struct {
	streamService *service.Stream
	template      []string
	childState    string
	withChild     []datatype.Map
}

// NewStepNewChild returns a fully initialized StepNewChild record
func NewStepNewChild(streamService *service.Stream, stepInfo datatype.Map) StepNewChild {
	return StepNewChild{
		streamService: streamService,
		template:      stepInfo.GetSliceOfString("template"),
		childState:    stepInfo.GetString("childState"),
		withChild:     stepInfo.GetSliceOfMap("withChild"),
	}
}

func (step StepNewChild) Get(buffer io.Writer, renderer *Renderer) error {

	templateID := renderer.ctx.QueryParam("templateId")

	// If no template has been designated, then choose a template.
	if templateID == "" {
		template, _ := renderer.template.HTMLTemplate("stream-create")

		if err := template.Execute(buffer, renderer); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Get", "Error executing template")
		}
		return nil
	}
	return nil
}

func (step StepNewChild) Post(buffer io.Writer, renderer *Renderer) error {

	templateID := renderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.template) > 0 {

		if templateID == "" {
			templateID = step.template[0]
		} else if !compare.Contains(step.template, templateID) {
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
	childRenderer, err := renderer.newRenderer(&child, "edit")

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating renderer", child)
	}

	// If there is an "init" step for the child's template, then execute it now
	if action, ok := template.Action("init"); ok {
		if err := DoPipeline(&childRenderer, buffer, action.Steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute 'init' action on child")
		}
	}

	if child.IsNew() {
		if err := step.streamService.Save(&child, "Created"); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error saving child stream to database")
		}
	}

	if err := DoPipeline(&childRenderer, buffer, step.withChild, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute action steps on child")
	}

	return nil
}
