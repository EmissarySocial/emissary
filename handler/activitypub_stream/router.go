package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/router"
)

// streamRouter defines the package-level router for stream/ActivityPub requests
var streamRouter = router.New[Context]()

// Context includes all of the necessary objects to handle an ActivityPub request
type Context struct {
	factory *service.Factory
	session data.Session
	stream  *model.Stream
	actor   *model.StreamActor
}

func (context Context) ActivityPubActor() (outbox.Actor, error) {

	return context.factory.Stream().ActivityPubActor(context.session, context.stream.StreamID)
}
