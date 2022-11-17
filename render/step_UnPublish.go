package render

import (
	"io"
	"time"
)

// StepUnPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepUnPublish struct{}

func (step StepUnPublish) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepUnPublish) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepUnPublish) Post(renderer Renderer) error {
	streamRenderer := renderer.(*Stream)
	streamRenderer.stream.PublishDate = time.Now().UnixMilli()
	return nil
}
