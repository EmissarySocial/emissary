package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepCreateChild is an action that can add new sub-streams to the domain.
type StepCreateChild struct {
	streamService *service.Stream
	template      []string
	childState    string
	withChild     []datatype.Map
}

// NewStepCreateChild returns a fully initialized StepCreateChild record
func NewStepCreateChild(streamService *service.Stream, stepInfo datatype.Map) StepCreateChild {
	return StepCreateChild{
		streamService: streamService,
		template:      stepInfo.GetSliceOfString("template"),
		childState:    stepInfo.GetString("childState"),
		withChild:     stepInfo.GetSliceOfMap("withChild"),
	}
}

func (step StepCreateChild) Get(buffer io.Writer, renderer *Renderer) error {

	templateID := renderer.ctx.QueryParam("templateId")

	// If no template has been designated, then choose a template.
	if templateID == "" {
		template, _ := renderer.template.HTMLTemplate("stream-create")

		if err := template.Execute(buffer, renderer); err != nil {
			return derp.Wrap(err, "ghost.render.StepCreateChild.Get", "Error executing template")
		}
		return nil
	}
	return nil
}

func (step StepCreateChild) Post(buffer io.Writer, renderer *Renderer) error {

	// Retrieve formData from request body
	var formData struct {
		TemplateID string `form:"templateId"`
	}

	if err := renderer.ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.render.StepCreateChild.Post", "Cannot bind form data")
	}

	// Validate that the requested template is allowed by this step
	if !compare.Contains(step.template, formData.TemplateID) {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepCreateChild.Post", "Invalid Template", formData.TemplateID)
	}

	// Create new child stream
	var child model.Stream

	authorization := getAuthorization(renderer.ctx)

	// Try to load the template that will be used

	template, err := step.streamService.Template(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepCreateChild.Post", "Undefined template", formData.TemplateID)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(renderer.stream.TemplateID) {
		return derp.Wrap(err, "ghost.render.StepCreateChild.Post", "Invalid template")
	}

	// Set Default Values
	child.ParentID = renderer.stream.StreamID
	child.StateID = step.childState
	child.AuthorID = authorization.UserID

	// Try to save the new child
	if err := step.streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.render.StepCreateChild.Post", "Error saving child")
	}

	// Success!  Send response to client
	renderer.ctx.Response().Header().Add("Hx-Redirect", "/"+child.Token)
	return nil
}
