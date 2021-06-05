package render

import (
	"net/http"

	"github.com/benpate/compare"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

//CreateStream is an action that can add new sub-streams to the domain.
type CreateStream struct {
	factory Factory
	model.ActionConfig
}

// NewAction_CreateStream returns a fully initialized CreateSubStream record
func NewAction_CreateStream(factory Factory, config model.ActionConfig) CreateStream {
	return CreateStream{
		factory:      factory,
		ActionConfig: config,
	}
}

type createStreamFormData struct {
	TemplateID string `form:"templateId"`
}

func (action CreateStream) Get(renderer Renderer) (string, error) {
	return "", nil
}

func (action CreateStream) Post(ctx steranko.Context, parent *model.Stream) error {

	// Retrieve formData from request body
	var formData createStreamFormData

	if err := ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Cannot bind form data")
	}

	// Validate that the requested template is allowed by this action
	if !compare.Contains(action.templateID(), formData.TemplateID) {
		return derp.New(derp.CodeBadRequestError, "ghost.render.CreateStream.Post", "Invalid Template", formData.TemplateID)
	}

	// Create new child stream
	var child model.Stream

	authorization := getAuthorization(&ctx)

	// Try to load the template that will be used

	streamService := action.factory.Stream()
	template, err := streamService.Template(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Undefined template", formData.TemplateID)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(parent.TemplateID) {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Invalid template")
	}

	// Set Default Values
	child.ParentID = parent.StreamID
	child.StateID = action.childState()
	child.AuthorID = authorization.UserID

	// Try to save the new child
	if err := streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.render.CreateStream.Post", "Error saving child")
	}

	// Success!  Send response to client
	ctx.Response().Header().Add("Hx-Redirect", "/"+child.Token)
	return ctx.NoContent(http.StatusOK)
}

// childState is a shortcut to the config value
func (action CreateStream) childState() string {
	return action.GetString("childState")
}

// childState is a shortcut to the config value
func (action CreateStream) templateID() string {
	return action.GetString("templateId")
}
