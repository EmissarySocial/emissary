package activitypub_search

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/outbox"
)

// Context includes all of the necessary objects to handle an ActivityPub request
type Context struct {
	factory     *domain.Factory
	stream      *model.Stream
	searchQuery *model.SearchQuery
}

func (context Context) ActivityPubActor(withFollowers bool) (outbox.Actor, error) {
	searchQueryService := context.factory.SearchQuery()
	return searchQueryService.ActivityPubActor(context.searchQuery, withFollowers)
}
