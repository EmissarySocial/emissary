package render

import (
	"io"
	"time"

	"github.com/benpate/datatype"
)

// StepSetPublishDate represents an action-step that can update a stream's PublishDate with the current time.
type StepSetPublishDate struct{}

// NewStepSetPublishDate returns a fully initialized StepSetPublishDate object
func NewStepSetPublishDate(stepInfo datatype.Map) StepSetPublishDate {
	return StepSetPublishDate{}
}

// Get does not display any data
func (step StepSetPublishDate) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSetPublishDate) Post(buffer io.Writer, renderer Renderer) error {
	streamRenderer := renderer.(*Stream)
	streamRenderer.stream.PublishDate = time.Now().UnixMilli()
	return nil
}
