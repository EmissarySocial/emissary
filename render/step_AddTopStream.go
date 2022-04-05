package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepAddTopStream represents an action that can create top-level folders in the Domain
type StepAddTopStream struct {
	templateIDs   []string // List of valid templateIds that the new top-level stream could be
	withNewStream Pipeline // Pipeline of steps to take on the newly-created stream

	BaseStep
}

// NewStepAddTopStream returns a fully parsed StepAddTopStream object
func NewStepAddTopStream(stepInfo datatype.Map) (StepAddTopStream, error) {

	withNewStream, err := NewPipeline(stepInfo.GetSliceOfMap("with-new-stream"))

	if err != nil {
		return StepAddTopStream{}, derp.Wrap(err, "render.StepAddTopStream", "Invalid 'with-new-stream", stepInfo)
	}

	return StepAddTopStream{
		templateIDs:   stepInfo.GetSliceOfString("templateIds"),
		withNewStream: withNewStream,
	}, nil
}

func (step StepAddTopStream) Get(factory Factory, renderer Renderer, buffer io.Writer) error {
	modalAddStream(renderer.context().Response(), factory.Template(), buffer, renderer.URL(), "top", step.templateIDs)
	return nil
}

func (step StepAddTopStream) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAddTopStream.Post"

	// Collect prerequisites
	topLevelRenderer := renderer.(TopLevel)
	templateID := topLevelRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.templateIDs) > 0 {
		if templateID == "" {
			templateID = step.templateIDs[0]
		} else if !compare.Contains(step.templateIDs, templateID) {
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

	return finalizeAddStream(buffer, renderer.factory(), renderer.context(), &stream, template, step.withNewStream)
}
