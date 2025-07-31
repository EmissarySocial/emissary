package activitypub_search

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/outbox"
)

// Context includes all of the necessary objects to handle an ActivityPub request
type Context struct {
	factory     *domain.Factory
	session     data.Session
	stream      *model.Stream
	searchQuery *model.SearchQuery
}

func (context Context) ActivityPubActor() (outbox.Actor, error) {
	searchQueryService := context.factory.SearchQuery()
	return searchQueryService.ActivityPubActor(context.session, context.searchQuery.SearchQueryID)
}
