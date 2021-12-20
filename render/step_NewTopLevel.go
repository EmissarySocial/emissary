package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
)

// StepNewTopLevel represents an action that can create top-level folders in the Domain
type StepNewTopLevel struct {
	templateService *service.Template
	streamService   *service.Stream
	templateIDs     []string
	withNewStream   []datatype.Map
}

// NewStepNewTopLevel returns a fully parsed StepNewTopLevel object
func NewStepNewTopLevel(templateService *service.Template, streamService *service.Stream, config datatype.Map) StepNewTopLevel {

	return StepNewTopLevel{
		templateService: templateService,
		streamService:   streamService,
		templateIDs:     config.GetSliceOfString("templateIds"),
		withNewStream:   config.GetSliceOfMap("with-new-stream"),
	}
}

func (step StepNewTopLevel) Get(buffer io.Writer, renderer Renderer) error {
	modalNewChild(step.templateService, buffer, renderer.URL(), "top", step.templateIDs)
	return nil
}

func (step StepNewTopLevel) Post(buffer io.Writer, renderer Renderer) error {

	topLevelRenderer := renderer.(*TopLevel)
	templateID := topLevelRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.templateIDs) > 0 {

		if templateID == "" {
			templateID = step.templateIDs[0]
		} else if !compare.Contains(step.templateIDs, templateID) {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepNewTopLevel.Post", "Cannot create new template of this kind", templateID)
		}
	}

	new, template, err := step.streamService.NewTopLevel(templateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewTopLevel.Post", "Error creating TopLevel stream", templateID)
	}

	// Set stream defaults
	authorization := getAuthorization(topLevelRenderer.ctx)
	new.AuthorID = authorization.UserID
	newStream, err := NewStream(topLevelRenderer.factory, topLevelRenderer.ctx, template, &new, "view")

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepNewTopLevel.Post", "Error creating renderer", new)
	}

	// If there is an "init" step for the new stream's template, then execute it now
	if action, ok := template.Action("init"); ok {
		if err := DoPipeline(topLevelRenderer.factory, &newStream, buffer, action.Steps, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewTopLevel.Post", "Unable to execute 'init' action on new stream")
		}
	}

	// Execute additional steps on new stream (from schema.json)
	if err := DoPipeline(topLevelRenderer.factory, newStream, buffer, step.withNewStream, ActionMethodPost); err != nil {
		return derp.Wrap(err, "ghost.render.StepNewTopLevel.Post", "Error executing steps on new stream")
	}

	// If the pipeline above did not already save the new stream, then save it to the database.
	if newStream.stream.IsNew() {
		if err := step.streamService.Save(newStream.stream, ""); err != nil {
			return derp.Wrap(err, "ghost.render.StepNewTopLevel.Post", "Error saving new steram", newStream.stream)
		}
	}

	return nil
}
