package tasks

import (
	"bytes"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
)

type ReceiveWebMention struct {
	streamService  *service.Stream
	mentionService *service.Mention
	source         string
	target         string
}

func NewReceiveWebMention(streamService *service.Stream, mentionService *service.Mention, source string, target string) ReceiveWebMention {
	return ReceiveWebMention{
		streamService:  streamService,
		mentionService: mentionService,
		source:         source,
		target:         target,
	}
}

func (task ReceiveWebMention) Run() error {

	const location = "tasks.ReceiveWebMention.Run"

	var content bytes.Buffer

	stream := model.NewStream()

	// Try to load the stream, to validate that the mention points to an eligible stream
	if err := task.streamService.LoadByURL(task.target, &stream); err != nil {
		return derp.Wrap(err, location, "Cannot load stream", task.target)
	}

	// Validate that the WebMention source actually links to the stream
	if err := task.mentionService.Verify(task.source, task.target, &content); err != nil {
		return derp.Wrap(err, location, "Source does not link to target", task.source, task.target)
	}

	// Parse the WebMention source into a Mention object
	mention := task.mentionService.ParseMicroformats(&content, task.target)

	// Try to save the mention to the database
	if err := task.mentionService.Save(&mention, "Created"); err != nil {
		return derp.Wrap(err, location, "Error saving mention")
	}

	return nil
}
