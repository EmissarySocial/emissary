package render

import (
	"io"
	"time"
)

// StepSetPublishDate represents an action-step that can update a stream's PublishDate with the current time.
type StepSetPublishDate struct{}

func (step StepSetPublishDate) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepSetPublishDate) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepSetPublishDate) Post(renderer Renderer, _ io.Writer) error {
	streamRenderer := renderer.(*Stream)
	streamRenderer.stream.PublishDate = time.Now().UnixMilli()
	return nil
}
