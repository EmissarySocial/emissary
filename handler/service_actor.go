package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

func GetServiceActor(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetServiceActor"

	return func(ctx echo.Context) error {

		// Retrieve the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error getting server factory")
		}

		// Retrieve the domain and Public Key
		domainService := factory.Domain()
		publicKeyPEM, err := domainService.PublicKeyPEM()

		if err != nil {
			return derp.Wrap(err, location, "Error getting public key PEM")
		}

		actorID := domainService.ActorID()

		// Return the result as a JSON-LD document
		result := map[string]any{
			vocab.AtContext:                []any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
			vocab.PropertyType:             vocab.ActorTypeService,
			vocab.PropertyID:               actorID,
			vocab.PropertyName:             domainService.Hostname(),
			vocab.PropertyFollowing:        actorID + "/following",
			vocab.PropertyFollowers:        actorID + "/followers",
			vocab.PropertyLiked:            actorID + "/liked",
			vocab.PropertyOutbox:           actorID + "/outbox",
			vocab.PropertyInbox:            actorID + "/inbox",
			vocab.PropertyTootDiscoverable: false,
			vocab.PropertyTootIndexable:    false,

			vocab.PropertyPublicKey: map[string]any{
				vocab.PropertyID:           domainService.PublicKeyID(),
				vocab.PropertyOwner:        actorID,
				vocab.PropertyPublicKeyPEM: publicKeyPEM,
			},
		}

		return ctx.JSON(200, result)
	}
}

// GetEmptyCollection returns an empty collection
func GetEmptyCollection(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		result := mapof.Any{
			vocab.AtContext:          vocab.ContextTypeActivityStreams,
			vocab.PropertyType:       vocab.CoreTypeOrderedCollection,
			vocab.PropertyTotalItems: 0,
			vocab.PropertyItems:      []any{},
		}

		return ctx.JSON(http.StatusOK, result)
	}
}

// PostServiceActor_Inbox does not take any actions, but only logs the request
// IF logger is in Debug or Trace mode.
func PostServiceActor_Inbox(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		if zerolog.GlobalLevel() > zerolog.DebugLevel {
			return ctx.NoContent(http.StatusOK)
		}

		// Try to read/dump the Request body
		// body, err := io.ReadAll(ctx.Request().Body)
		// log.Trace().Msg(string(body))

		// Return no content
		return ctx.NoContent(http.StatusOK)
	}
}
