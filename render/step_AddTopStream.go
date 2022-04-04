package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/service"
	"github.com/whisperverse/whisperverse/singleton"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepAddTopStream represents an action that can create top-level folders in the Domain
type StepAddTopStream struct {
	templateService *singleton.Template
	streamService   *service.Stream
	templateIDs     []string
	withNewStream   []datatype.Map
}

// NewStepAddTopStream returns a fully parsed StepAddTopStream object
func NewStepAddTopStream(templateService *singleton.Template, streamService *service.Stream, config datatype.Map) StepAddTopStream {

	return StepAddTopStream{
		templateService: templateService,
		streamService:   streamService,
		templateIDs:     config.GetSliceOfString("templateIds"),
		withNewStream:   config.GetSliceOfMap("with-new-stream"),
	}
}

func (step StepAddTopStream) Get(buffer io.Writer, renderer Renderer) error {
	modalAddStream(renderer.context().Response(), step.templateService, buffer, renderer.URL(), "top", step.templateIDs)
	return nil
}

func (step StepAddTopStream) Post(buffer io.Writer, renderer Renderer) error {

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
	template, err := step.templateService.Load(templateID)

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
