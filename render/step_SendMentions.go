package render

import (
	"io"

	"github.com/EmissarySocial/emissary/tasks"
	"github.com/benpate/derp"
)

// StepSendMentions represents an action-step that can update the custom data stored in a Stream
type StepSendMentions struct{}

func (step StepSendMentions) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepSendMentions) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSendMentions) Post(renderer Renderer) error {

	const location = "render.StepSendMentions.Post"

	html, err := renderer.View("view")

	if err != nil {
		return derp.Wrap(err, location, "Error rendering HTML", html)
	}

	// Push the "send" task onto the queue
	mentionService := renderer.factory().Mention()
	queue := renderer.factory().Queue()
	task := tasks.NewSendWebMention(mentionService, renderer.Permalink(), string(html))
	queue.Run(task)

	// Silence is Awesome
	return nil
}
