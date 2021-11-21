package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepTopFolderCreate represents an action that can create top-level folders in the Domain
type StepTopFolderCreate struct {
	streamService *service.Stream
	parent        string
	templateID    string
}

// NewStepTopFolderCreate returns a fully parsed StepTopFolderCreate object
func NewStepTopFolderCreate(streamService *service.Stream, config datatype.Map) StepTopFolderCreate {

	return StepTopFolderCreate{
		streamService: streamService,
		parent:        config.GetString("parent"),
		templateID:    config.GetString("templateId"),
	}
}

type createTopStreamFormData struct {
	Parent     string `form:"parent"`
	TemplateID string `form:"templateId"`
}

func (step StepTopFolderCreate) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepTopFolderCreate) Post(buffer io.Writer, renderer *Renderer) error {

	// Retrieve formData from request body
	var formData createTopStreamFormData
	var child model.Stream

	if err := renderer.ctx.Bind(&formData); err != nil {
		return derp.Wrap(err, "ghost.render.StepTopFolderCreate.Post", "Cannot bind form data")
	}

	// Try to load the template
	template, err := step.streamService.Template(formData.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepTopFolderCreate.Post", "Invalid template")
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
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepTopFolderCreate.Post", "Cannot place template on top", formData.TemplateID)
		}

	} else {

		// Validate the parent exists, and that we cna put a folder here....
		var parent model.Stream

		// Try to load the parent stream
		if err := step.streamService.LoadByToken(formData.Parent, &parent); err != nil {
			return derp.Wrap(err, "ghost.render.StepTopFolderCreate.Post", "Error loading parent stream")
		}

		// Confirm that this Template can be a child of the parent Template
		if !template.CanBeContainedBy(parent.TemplateID) {
			return derp.Wrap(err, "ghost.render.StepTopFolderCreate.Post", "Template cannot be placed in parent")
		}

		// Everything checks out.  Assign the child to the parent
		child.ParentID = parent.StreamID
	}

	// Save the new child
	if err := step.streamService.Save(&child, "created"); err != nil {
		return derp.Wrap(err, "ghost.render.StepTopFolderCreate.Post", "Error saving child")
	}

	// Success! Write response to client
	renderer.ctx.Response().Header().Add("HX-Redirect", "/"+child.Token)
	return nil
}
