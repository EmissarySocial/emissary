package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateTopStream struct {
	streamService *service.Stream
	parent        string
	templateID    string
}

func NewCreateTopStream(streamService *service.Stream, config datatype.Map) CreateTopStream {

	return CreateTopStream{
		streamService: streamService,
		parent:        config.GetString("parent"),
		templateID:    config.GetString("templateId"),
	}
}

type createTopStreamFormData struct {
	Parent     string `form:"parent"`
	TemplateID string `form:"templateId"`
}

func (step CreateTopStream) Get(renderer *Renderer) error {
	return nil
}

func (step CreateTopStream) Post(renderer *Renderer) error {

	// Retrieve formData from request body
	var formData createTopStreamFormData
	var child model.Stream

	if err := renderer.ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.render.CreateTopStream.Post", "Cannot bind form data")
	}

	// Try to load the template
	template, err := step.streamService.Template(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.CreateTopStream.Post", "Invalid template")
	}

	authorization := getAuthorization(renderer.ctx)

	// Create new child stream
	child.TemplateID = "folder"
	child.StateID = "default"
	child.AuthorID = authorization.UserID

	// FIgure out where to put the new child
	if formData.Parent == "top" {
		child.ParentID = primitive.NilObjectID

		// Verify that this template can live on the top level
		if !template.CanBeContainedBy("top") {
			return derp.New(derp.CodeBadRequestError, "ghost.render.CreateTopStream.Post", "Cannot place template on top", formData.TemplateID)
		}

	} else {

		// Validate the parent exists, and that we cna put a folder here....
		var parent model.Stream

		// Try to load the parent stream
		if err := step.streamService.LoadByToken(formData.Parent, &parent); err != nil {
			return derp.Wrap(err, "ghost.render.CreateTopStream.Post", "Error loading parent stream")
		}

		// Confirm that this Template can be a child of the parent Template
		if !template.CanBeContainedBy(parent.TemplateID) {
			return derp.Wrap(err, "ghost.render.CreateTopStream.Post", "Template cannot be placed in parent")
		}

		// Everything checks out.  Assign the child to the parent
		child.ParentID = parent.StreamID
	}

	// Save the new child
	if err := step.streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.render.CreateTopStream.Post", "Error saving child")
	}

	// Success! Write response to client
	renderer.ctx.Response().Header().Add("HX-Redirect", "/"+child.Token)
	return renderer.ctx.NoContent(http.StatusOK)
}
