package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
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

	authorization := getAuthorization(renderer.ctx)

	// Try to load the template that will be used

	template, err := step.streamService.Template(templateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Undefined template", templateID)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(renderer.stream.TemplateID) {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Invalid template")
	}

	// Set Default Values
	child := model.NewStream()
	child.ParentID = renderer.stream.StreamID
	child.StateID = step.childState
	child.TemplateID = templateID
	child.AuthorID = authorization.UserID
	child.Token = child.StreamID.Hex()

	childRenderer, err := renderer.newRenderer(&child, "view")

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Error creating renderer", child)
	}

	// If there is an "init" step for the child's template, then execute it now
	if action, ok := template.Action("init"); ok {
		if err := DoPipeline(&childRenderer, buffer, action.Steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute 'init' action on child")
		}
	}

	if err := DoPipeline(&childRenderer, buffer, step.withChild, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewChild.Post", "Unable to execute action steps on child")
	}

	return nil
}
