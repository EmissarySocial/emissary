package render

import (
	"io"

	"github.com/benpate/compare"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/service"
)

// StepAddTopStream represents an action that can create top-level folders in the Domain
type StepAddTopStream struct {
	templateService *service.Template
	streamService   *service.Stream
	templateIDs     []string
	withNewStream   []datatype.Map
}

// NewStepAddTopStream returns a fully parsed StepAddTopStream object
func NewStepAddTopStream(templateService *service.Template, streamService *service.Stream, config datatype.Map) StepAddTopStream {

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

	topLevelRenderer := renderer.(TopLevel)
	templateID := topLevelRenderer.ctx.QueryParam("templateId")

	// If there is a list of eligible templates, then guarantee that the new template is in the list.
	if len(step.templateIDs) > 0 {

		if templateID == "" {
			templateID = step.templateIDs[0]
		} else if !compare.Contains(step.templateIDs, templateID) {
			return derp.New(derp.CodeBadRequestError, "whisper.render.StepAddTopStream.Post", "Cannot create new template of this kind", templateID)
		}
	}

	topLevelStream, template, err := step.streamService.NewTopLevel(templateID)

	if err != nil {
		return derp.Wrap(err, "whisper.render.StepAddTopStream.Post", "Error creating TopLevel stream", templateID)
	}

	return finalizeAddStream(buffer, renderer.factory(), renderer.context(), &topLevelStream, template, step.withNewStream)
}
