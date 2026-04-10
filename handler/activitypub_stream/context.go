package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/outbox"
)


func (context Context) ActivityPubActor() (outbox.Actor, error) {

	return context.factory.Stream().ActivityPubActor(context.session, context.stream.StreamID)
}
