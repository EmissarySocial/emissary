package render

import (
	"fmt"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tasks"
	"github.com/EmissarySocial/emissary/tools/domain"
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

	// RULE: Don't send mentions if the server is local
	if domain.IsLocalhost(renderer.Hostname()) {
		fmt.Println("Skipping mentions because hostname is localhost")
		return nil
	}

	// RULE: Don't send mentions on items that require login
	if !renderer.UserCan(model.MagicRoleAnonymous) {
		fmt.Println("Skipping mentions because user is not anonymous")
		return nil
	}

	// Get full HTML for this Stream
	html, err := renderer.View("view")

	if err != nil {
		return derp.Wrap(err, location, "Error rendering HTML", html)
	}

	// Push the "send webmention" task onto the queue
	mentionService := renderer.factory().Mention()
	queue := renderer.factory().Queue()
	task := tasks.NewSendWebMention(mentionService, renderer.Permalink(), string(html))
	queue.Run(task)

	// Hello darkness, my old friend...
	return nil
}
