package render

import (
	"fmt"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tasks"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/derp"
)

// StepPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepPublish struct {
	Role string
}

func (step StepPublish) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepPublish) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepPublish) Post(renderer Renderer) error {

	const location = "render.StepPublish.Post"

	streamRenderer := renderer.(*Stream)

	if err := step.publish(streamRenderer); err != nil {
		return derp.Wrap(err, location, "Error publishing stream")
	}

	if err := step.sendWebMentions(streamRenderer); err != nil {
		return derp.Wrap(err, location, "Error sending web mentions")
	}

	return nil
}

// Post updates the stream with the current date as the "PublishDate"
func (step StepPublish) publish(renderer *Stream) error {

	const location = "render.StepPublish.publish"

	// Require that the user is signed in to perform this action
	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError(location, "User is not authenticated", nil)
	}

	// Use the publisher service to execute publishing rules
	stream := renderer.stream

	publisherService := renderer.factory().Publisher()
	publisherService.Publish(stream, renderer.AuthenticatedID())

	step.sendWebMentions(renderer)

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepPublish) sendWebMentions(renderer *Stream) error {

	const location = "render.StepPublish.sendWebMentions"

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
