package render

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"willnorris.com/go/webmention"
)

// StepPublish represents an action-step that can update a stream's PublishDate with the current time.
type StepPublish struct {
	Mentions []string
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

	// Push the "send webmention" task onto the queue
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

	outobxService := renderer.factory().Outbox()
	outobxService.Publish(renderer.AuthenticatedID(), stream)

	return nil
}

// sendWebMentionds sends WebMention updates to external websites that are
// mentioned in this stream.  This is here (and not in the outbox service)
// because we need to render the content in order to discover outbound links.
func (step StepPublish) sendWebMentions(renderer *Stream) error {

	var bodyReader bytes.Buffer

	factory := renderer.factory()
	schema := renderer.schema()

	// Collect all content fields from the schema
	for _, fieldName := range step.Mentions {
		if content, err := schema.Get(renderer.stream, fieldName); err == nil {
			bodyReader.WriteString(convert.String(content))
		}
	}

	// Discover all webmention links in the content
	links, err := webmention.DiscoverLinksFromReader(&bodyReader, renderer.Permalink(), "")

	if err != nil {
		return derp.Wrap(err, "mention.SendWebMention.Run", "Error discovering webmention links", renderer.Permalink())
	}

	// If no links, peace out.
	if len(links) == 0 {
		return nil
	}

	// Add background tasks to TRY sending webmentions to every link we found
	queue := factory.Queue()

	for _, link := range links {
		queue.Run(service.NewTaskSendWebMention(renderer.Permalink(), link))
	}

	// Accept your success with grace.
	return nil
}
