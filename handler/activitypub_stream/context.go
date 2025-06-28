package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/outbox"
)

// Context includes all of the necessary objects to handle an ActivityPub request
type Context struct {
	factory *domain.Factory
	stream  *model.Stream
	actor   *model.StreamActor
}

func (context Context) ActivityPubActor() (outbox.Actor, error) {

	return context.factory.Stream().ActivityPubActor(context.stream.StreamID)
}
