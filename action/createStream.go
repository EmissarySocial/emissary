package action

import (
	"net/http"

	"github.com/benpate/compare"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
)

//CreateStream is an action that can add new sub-streams to the domain.
type CreateStream struct {
	config        model.ActionConfig
	streamService *service.Stream
}

// NewAction_CreateStream returns a fully initialized CreateSubStream record
func NewAction_CreateStream(config model.ActionConfig, streamService *service.Stream) CreateStream {
	return CreateStream{
		config:        config,
		streamService: streamService,
	}
}

type createStreamFormData struct {
	TemplateID string `form:"templateId"`
}

func (action *CreateStream) Get(ctx steranko.Context, parent *model.Stream) error {
	return nil
}

func (action *CreateStream) Post(ctx steranko.Context, parent *model.Stream) error {

	// Retrieve formData from request body
	var formData createStreamFormData

	if err := ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Cannot bind form data")
	}

	// Validate that the requested template is allowed by this action
	if !compare.Contains(action.TemplateID, formData.TemplateID) {
		return derp.New(derp.CodeBadRequestError, "ghost.action.CreateStream.Post", "Invalid Template", formData.TemplateID)
	}

	// Create new child stream
	var child model.Stream

	authorization := getAuthorization(ctx)

	// Try to load the template that will be used
	template, err := action.streamService.Template(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Undefined template", formData.TemplateID)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(parent.TemplateID) {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Invalid template")
	}

	// Set Default Values
	child.ParentID = parent.StreamID
	child.StateID = action.ChildStateID
	child.AuthorID = authorization.UserID

	// Try to save the new child
	if err := action.streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Error saving child")
	}

	// Success!  Send response to client
	ctx.Response().Header().Add("Hx-Redirect", "/"+child.Token)
	return ctx.NoContent(http.StatusOK)
}

// Config returns the configuration information for this action
func (action *CreateStream) Config() model.ActionConfig {
	return action.config
}
