package activitypub_search

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/steranko"
)

func GetFollowersCollection(ctx *steranko.Context, factory *domain.Factory, stream *model.Stream, searchQuery *model.SearchQuery) error {
	return nil
}
