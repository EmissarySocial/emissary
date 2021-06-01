package action

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateFolder struct {
	ChildStateID string
	TemplateID   string
	CommonInfo
}

type createFolderFormData struct {
	Parent     string `form:"parent"`
	TemplateID string `form:"templateId"`
}

func (action CreateFolder) Get(ctx steranko.Context, factory *domain.Factory, _ *model.Stream) error {
	return nil
}

func (action CreateFolder) Put(ctx steranko.Context, factory *domain.Factory, _ *model.Stream) error {

	// Retrieve formData from request body
	var formData createFolderFormData
	var child model.Stream

	if err := ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.action.CreateFolder.Post", "Cannot bind form data")
	}

	// Get required services
	templateService := factory.Template()
	streamService := factory.Stream()
	authorization := getAuthorization(ctx)

	template, err := templateService.Load(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.action.CreateFolder.Post", "Invalid template")
	}

	// Create new child stream
	child.TemplateID = "folder"
	child.StateID = "default"
	child.AuthorID = authorization.UserID

	// FIgure out where to put the new child
	if formData.Parent == "top" {
		child.ParentID = primitive.NilObjectID

		// Verify that this template can live on the top level
		if !template.CanBeContainedBy("top") {
			return derp.New(derp.CodeBadRequestError, "ghost.action.CreateFolder.Post", "Cannot place template on top", formData.TemplateID)
		}

	} else {

		// Validate the parent exists, and that we cna put a folder here....
		var parent model.Stream

		// Try to load the parent stream
		if err := streamService.LoadByToken(formData.Parent, &parent); err != nil {
			return derp.Wrap(err, "ghost.action.CreateFolder.Post", "Error loading parent stream")
		}

		// Confirm that this Template can be a child of the parent Template
		if !template.CanBeContainedBy(parent.TemplateID) {
			return derp.Wrap(err, "ghost.service.Stream.NewWithTemplate", "Invalid template")
		}

		// Everything checks out.  Assign the child to the parent
		child.ParentID = parent.StreamID
	}

	// Save the new child
	if err := streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.action.CreateFolder.Post", "Error saving child")
	}

	// Success! Write response to client
	ctx.Response().Header().Add("HX-Redirect", "/"+child.Token)
	return ctx.NoContent(http.StatusOK)
}
