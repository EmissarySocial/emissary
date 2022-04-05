package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/model/step"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepAddTopStream represents an action that can create top-level folders in the Domain
type StepAddTopStream struct {
	TemplateIDs   []string    // List of valid templateIds that the new top-level stream could be
	WithNewStream []step.Step // Pipeline of steps to take on the newly-created stream
}

func (step StepAddTopStream) Get(renderer Renderer, buffer io.Writer) error {
	factory := renderer.factory()
	modalAddStream(renderer.context().Response(), factory.Template(), buffer, renderer.URL(), "top", step.TemplateIDs)
	return nil
}

func (step StepAddTopStream) Post(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAddTopStream.Post"

	// Collect prerequisites
	factory := renderer.factory()
	topLevelRenderer := renderer.(TopLevel)
	templateID := topLevelRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.TemplateIDs) > 0 {
		if templateID == "" {
			templateID = step.TemplateIDs[0]
		} else if !compare.Contains(step.TemplateIDs, templateID) {
			return derp.New(derp.CodeBadRequestError, location, "Cannot create new template of this kind", templateID)
		}
	}

	// Try to load the template for the new stream
	template, err := factory.Template().Load(templateID)

	if err != nil {
		return derp.Wrap(err, location, "Cannot find template")
	}

	// Verify that the template can be placed at the top level
	if !template.CanBeContainedBy("top") {
		return derp.New(derp.CodeInternalError, location, "Template cannot be placed at top level", templateID)
	}

	// Create new top-level stream
	stream := model.NewStream()
	stream.ParentID = primitive.NilObjectID
	stream.ParentIDs = make([]primitive.ObjectID, 0)
	stream.TemplateID = templateID

	// TODO: sort order?

	return finalizeAddStream(buffer, renderer.factory(), renderer.context(), &stream, template, step.WithNewStream)
}
