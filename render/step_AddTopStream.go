package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/compare"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepAddTopStream represents an action that can create top-level folders in the Domain
type StepAddTopStream struct {
	Title         string
	TemplateIDs   []string    // List of valid templateIds that the new top-level stream could be
	WithNewStream []step.Step // Pipeline of steps to take on the newly-created stream
}

func (step StepAddTopStream) Get(renderer Renderer, buffer io.Writer) error {
	factory := renderer.factory()
	modalAddStream(renderer.context().Response(), factory.Template(), factory.Icons(), step.Title, buffer, renderer.URL(), "top", step.TemplateIDs)
	return nil
}

func (step StepAddTopStream) UseGlobalWrapper() bool {
	return false
}

func (step StepAddTopStream) Post(renderer Renderer) error {

	const location = "render.StepAddTopStream.Post"

	// Collect prerequisites
	factory := renderer.factory()
	context := renderer.context()
	templateID := context.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.TemplateIDs) > 0 {
		if templateID == "" {
			templateID = step.TemplateIDs[0]
		} else if !compare.Contains(step.TemplateIDs, templateID) {
			return derp.NewBadRequestError(location, "Cannot create new template of this kind", templateID)
		}
	}

	// Try to load the template for the new stream
	template, err := factory.Template().Load(templateID)

	if err != nil {
		return derp.Wrap(err, location, "Cannot find template")
	}

	// Verify that the template can be placed at the top level
	if !template.CanBeContainedBy("top") {
		return derp.NewInternalError(location, "Template cannot be placed at top level", templateID)
	}

	// Create new top-level stream
	stream := model.NewStream()
	stream.ParentID = primitive.NilObjectID
	stream.TopLevelID = stream.StreamID.Hex()
	stream.TemplateID = templateID

	// TODO: MEDIUM: sort order?

	return finalizeAddStream(renderer.factory(), renderer.context(), &stream, template, step.WithNewStream)
}
