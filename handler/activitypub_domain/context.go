package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/outbox"
)

// Context includes all of the necessary objects to handle an ActivityPub request
type Context struct {
	factory *service.Factory
	session data.Session
}

func (context Context) ActivityPubActor() (outbox.Actor, error) {
	return context.factory.SearchDomain().ActivityPubActor(context.session)
}
