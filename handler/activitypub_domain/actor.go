package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

func GetJSONLD(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.activitypub_domain.GetJSONLD"

	// Retrieve the domain and Public Key
	domainService := factory.Domain()
	publicKeyPEM, err := domainService.PublicKeyPEM()

	if err != nil {
		return derp.Wrap(err, location, "Error getting public key PEM")
	}

	searchDomainService := factory.SearchDomain()
	actorID := searchDomainService.ActivityPubURL()

	// Build the actor description
	summary := `This is an automated search query on the server: ` + factory.Hostname() + ` that announces new search results as they are received.  <a href="` + searchDomainService.ActivityPubProfileURL() + `">View the full collection on ` + factory.Hostname() + `</a>.`

	// Return the result as a JSON-LD document
	result := map[string]any{
		vocab.AtContext:                 []any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
		vocab.PropertyType:              vocab.ActorTypeService,
		vocab.PropertyID:                searchDomainService.ActivityPubURL(),
		vocab.PropertyURL:               searchDomainService.ActivityPubProfileURL(),
		vocab.PropertyPreferredUsername: searchDomainService.ActivityPubUsername(),
		vocab.PropertyName:              searchDomainService.ActivityPubName(),
		vocab.PropertySummary:           summary,
		vocab.PropertyInbox:             searchDomainService.ActivityPubInboxURL(),
		vocab.PropertyOutbox:            searchDomainService.ActivityPubOutboxURL(),
		vocab.PropertyFollowers:         searchDomainService.ActivityPubFollowersURL(),
		vocab.PropertyFollowing:         searchDomainService.ActivityPubFollowingURL(),
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
