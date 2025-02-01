package activitypub_search

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/steranko"
)

func GetOutboxCollection(ctx *steranko.Context, factory *domain.Factory, template *model.Template, stream *model.Stream, searchQuery *model.SearchQuery) error {
	searchQueryService := factory.SearchQuery()
	collectionID := searchQueryService.ActivityPubOutboxURL(searchQuery)
	result := activitypub.Collection(collectionID)
	return ctx.JSON(http.StatusOK, result)
}
