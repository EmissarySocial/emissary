package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepShareReplyPolicy is a Step that stamps the Stream's ReplyPolicy on the Streams in its User's outbox
// that use one Template.
type StepShareReplyPolicy struct {
	TemplateID string
}

// Get does nothing. Implements the Step interface.
func (step StepShareReplyPolicy) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post copies the Stream's ReplyPolicy onto its sibling outbox Streams
func (step StepShareReplyPolicy) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepShareReplyPolicy.Post"

	streamBuilder, ok := builder.(Stream)

	if !ok {
		return Halt().WithError(derp.Internal(location, "share-reply-policy requires a Stream builder"))
	}

	stream := streamBuilder._stream

	// RULE: Only a Stream in a User's outbox shares with that outbox
	if stream.ParentTemplateID != "" {
		return Continue()
	}

	// RULE: Only the outbox's own User may change its Streams
	if stream.ParentID != builder.AuthenticatedID() {
		return Halt().WithError(derp.Forbidden(location, "Only the outbox's owner may share its reply policy", stream.ParentID))
	}

	streamService := builder.factory().Stream()

	if err := streamService.SetOutboxReplyPolicy(builder.session(), stream.ParentID, step.TemplateID, stream.ReplyPolicy); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Sharing reply policy", step.TemplateID))
	}

	return Continue()
}
