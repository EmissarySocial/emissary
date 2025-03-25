package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/hannibal/outbox"
)

// Context includes all of the necessary objects to handle an ActivityPub request
type Context struct {
	factory *domain.Factory
}

func (context Context) ActivityPubActor(withFollowers bool) (outbox.Actor, error) {
	searchDomainService := context.factory.SearchDomain()
	return searchDomainService.ActivityPubActor(withFollowers)
}
