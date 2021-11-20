package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// CreateStream is an action that can add new sub-streams to the domain.
type CreateStream struct {
	streamService *service.Stream
	childState    string
	templateID    string
}

// NewCreateStream returns a fully initialized CreateSubStream record
func NewCreateStream(streamService *service.Stream, command datatype.Map) CreateStream {
	return CreateStream{
		streamService: streamService,
		childState:    command.GetString("childState"),
		templateID:    command.GetString("tempalteId"),
	}
}

type createStreamFormData struct {
	TemplateID string `form:"templateId"`
}

func (step CreateStream) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step CreateStream) Post(buffer io.Writer, renderer *Renderer) error {

	// Retrieve formData from request body
	var formData createStreamFormData

	if err := renderer.ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Cannot bind form data")
	}

	// Validate that the requested template is allowed by this step
	if !compare.Contains(step.templateID, formData.TemplateID) {
		return derp.New(derp.CodeBadRequestError, "ghost.render.CreateStream.Post", "Invalid Template", formData.TemplateID)
	}

	// Create new child stream
	var child model.Stream

	authorization := getAuthorization(renderer.ctx)

	// Try to load the template that will be used

	template, err := step.streamService.Template(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Undefined template", formData.TemplateID)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(renderer.stream.TemplateID) {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Invalid template")
	}

	// Set Default Values
	child.ParentID = renderer.stream.StreamID
	child.StateID = step.childState
	child.AuthorID = authorization.UserID

	// Try to save the new child
	if err := step.streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Error saving child")
	}

	// Success!  Send response to client
	renderer.ctx.Response().Header().Add("Hx-Redirect", "/"+child.Token)
	return nil
}
