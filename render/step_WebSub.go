package render

import (
	"io"

	"meow.tf/websub"
)

// StepWebSub represents an action-step that can render a Stream into HTML
type StepWebSub struct {
}

// Get renders the Stream HTML to the context
func (step StepWebSub) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepWebSub) UseGlobalWrapper() bool {
	return true
}

func (step StepWebSub) Post(renderer Renderer) error {
	outbox := renderer.factory().WebSubOutbox(renderer.objectID())
	handler := websub.New(outbox)
	handler.ServeHTTP(renderer.context().Response(), renderer.context().Request())
	return nil
}
