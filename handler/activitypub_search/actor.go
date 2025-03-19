package activitypub_search

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

func GetJSONLD(ctx *steranko.Context, factory *domain.Factory, template *model.Template, stream *model.Stream, searchQuery *model.SearchQuery) error {

	const location = "handler.activitypub_search.GetJSONLD"

	// Retrieve the domain and Public Key
	domainService := factory.Domain()
	publicKeyPEM, err := domainService.PublicKeyPEM()

	if err != nil {
		return derp.Wrap(err, location, "Error getting public key PEM")
	}

	searchQueryService := factory.SearchQuery()
	searchQueryID := searchQuery.SearchQueryID
	actorID := searchQueryService.ActivityPubURL(searchQueryID)

	// Return the result as a JSON-LD document
	result := map[string]any{
		vocab.AtContext:                 []any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
		vocab.PropertyType:              vocab.ActorTypeService,
		vocab.PropertyID:                searchQueryService.ActivityPubURL(searchQueryID),
		vocab.PropertyURL:               searchQueryService.ActivityPubProfileURL(searchQuery),
		vocab.PropertyPreferredUsername: searchQueryService.ActivityPubUsername(searchQueryID),
		vocab.PropertyName:              searchQueryService.ActivityPubName(searchQuery),
		vocab.PropertyInbox:             searchQueryService.ActivityPubInboxURL(searchQueryID),
		vocab.PropertyOutbox:            searchQueryService.ActivityPubOutboxURL(searchQueryID),
		vocab.PropertyFollowers:         searchQueryService.ActivityPubFollowersURL(searchQueryID),
		vocab.PropertyFollowing:         searchQueryService.ActivityPubFollowingURL(searchQueryID),
		vocab.PropertyTootDiscoverable:  false,
		vocab.PropertyTootIndexable:     false,

		vocab.PropertyPublicKey: map[string]any{
			vocab.PropertyID:           actorID + "#main-key",
			vocab.PropertyOwner:        actorID,
			vocab.PropertyPublicKeyPEM: publicKeyPEM,
		},
	}

	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(200, result)
}
