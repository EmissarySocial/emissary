package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepStreamCreate is an action that can add new sub-streams to the domain.
type StepStreamCreate struct {
	streamService *service.Stream
	childState    string
	templateID    string
}

// NewStepStreamCreate returns a fully initialized StepStreamCreate record
func NewStepStreamCreate(streamService *service.Stream, command datatype.Map) StepStreamCreate {
	return StepStreamCreate{
		streamService: streamService,
		childState:    command.GetString("childState"),
		templateID:    command.GetString("tempalteId"),
	}
}

type createStreamFormData struct {
	TemplateID string `form:"templateId"`
}

func (step StepStreamCreate) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepStreamCreate) Post(buffer io.Writer, renderer *Renderer) error {

	// Retrieve formData from request body
	var formData createStreamFormData

	if err := renderer.ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamCreate.Post", "Cannot bind form data")
	}

	// Validate that the requested template is allowed by this step
	if !compare.Contains(step.templateID, formData.TemplateID) {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamCreate.Post", "Invalid Template", formData.TemplateID)
	}

	// Create new child stream
	var child model.Stream

	authorization := getAuthorization(renderer.ctx)

	// Try to load the template that will be used

	template, err := step.streamService.Template(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamCreate.Post", "Undefined template", formData.TemplateID)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(renderer.stream.TemplateID) {
		return derp.Wrap(err, "ghost.render.StepStreamCreate.Post", "Invalid template")
	}

	// Set Default Values
	child.ParentID = renderer.stream.StreamID
	child.StateID = step.childState
	child.AuthorID = authorization.UserID

	// Try to save the new child
	if err := step.streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamCreate.Post", "Error saving child")
	}

	// Success!  Send response to client
	renderer.ctx.Response().Header().Add("Hx-Redirect", "/"+child.Token)
	return nil
}
