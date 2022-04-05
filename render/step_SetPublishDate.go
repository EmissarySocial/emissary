package render

import (
	"io"
	"time"

	"github.com/benpate/datatype"
)

// StepSetPublishDate represents an action-step that can update a stream's PublishDate with the current time.
type StepSetPublishDate struct {
	BaseStep
}

// NewStepSetPublishDate returns a fully initialized StepSetPublishDate object
func NewStepSetPublishDate(stepInfo datatype.Map) (StepSetPublishDate, error) {
	return StepSetPublishDate{}, nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSetPublishDate) Post(_ Factory, renderer Renderer, _ io.Writer) error {
	streamRenderer := renderer.(*Stream)
	streamRenderer.stream.PublishDate = time.Now().UnixMilli()
	return nil
}
