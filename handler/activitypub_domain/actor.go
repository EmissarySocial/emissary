package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

func GetJSONLD(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_domain.GetJSONLD"

	// Retrieve the domain and Public Key
	domainService := factory.Domain()
	publicKeyPEM, err := domainService.PublicKeyPEM(session)

	if err != nil {
		return derp.Wrap(err, location, "Error getting public key PEM")
	}

	searchDomainService := factory.SearchDomain()
	actorID := searchDomainService.ActivityPubURL()

	host := factory.Host()

	// Return the result as a JSON-LD document
	result := map[string]any{
		vocab.AtContext:                 []any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
		vocab.PropertyType:              vocab.ActorTypeService,
		vocab.PropertyID:                searchDomainService.ActivityPubURL(),
		vocab.PropertyURL:               searchDomainService.ActivityPubProfileURL(),
		vocab.PropertyPreferredUsername: searchDomainService.ActivityPubUsername(),
		vocab.PropertyName:              searchDomainService.ActivityPubName(),
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

		vocab.PropertyRedirectURI: []string{
			host + "/oauth/clients/import/redirect",
		},
	}

	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(200, result)
}
