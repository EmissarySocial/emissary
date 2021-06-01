package action

import (
	"net/http"

	"github.com/benpate/compare"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

type CreateStream struct {
	ChildStateID string
	TemplateID   []string
	CommonInfo
}

type createStreamFormData struct {
	TemplateID string `form:"templateId"`
}

func (action CreateStream) Get(ctx steranko.Context, factory *domain.Factory, parent *model.Stream) error {
	return nil
}

func (action CreateStream) Put(ctx steranko.Context, factory *domain.Factory, parent *model.Stream) error {

	// Retrieve formData from request body
	var formData createStreamFormData

	if err := ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Cannot bind form data")
	}

	// Validate that the requested template is allowed by this action
	if !compare.Contains(action.TemplateID, formData.TemplateID) {
		return derp.New(derp.CodeBadRequestError, "ghost.action.CreateStream.Post", "Invalid Template", formData.TemplateID)
	}

	// Get required services
	templateService := factory.Template()
	streamService := factory.Stream()

	// Create new child stream
	child := streamService.New()

	authorization := getAuthorization(ctx)

	// Try to load the template that will be used
	template, err := templateService.Load(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Undefined template", formData.TemplateID)
	}

	// Confirm that this Template can be a child of the parent Template
	if !template.CanBeContainedBy(parent.TemplateID) {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Invalid template")
	}

	// Set
	child.ParentID = parent.StreamID
	child.StateID = action.ChildStateID
	child.AuthorID = authorization.UserID

	// Try to save the new child
	if err := streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.action.CreateStream.Post", "Error saving child")
	}

	// Success!  Send response to client
	ctx.Response().Header().Add("Hx-Redirect", "/"+child.Token)
	return ctx.NoContent(http.StatusOK)
}
