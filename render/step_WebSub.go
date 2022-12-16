package render

import (
	"io"

	"github.com/benpate/websub"
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
	context := renderer.context()
	outbox := renderer.factory().WebSubOutbox(renderer.objectID())
	client := websub.New(outbox)
	client.ServeHTTP(context.Response(), context.Request())
	return nil
}
