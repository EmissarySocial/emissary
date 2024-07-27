package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/labstack/echo/v4"
)

func GetDomainActor(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetDomainActor"

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

		// Return the result as a JSON-LD document
		result := map[string]any{
			vocab.AtContext:    []string{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity},
			vocab.PropertyType: vocab.ActorTypeService,
			vocab.PropertyID:   domainService.ActorID(),
			vocab.PropertyName: domainService.Hostname(),
			vocab.PropertyPublicKey: map[string]any{
				vocab.PropertyID:           domainService.PublicKeyID(),
				vocab.PropertyOwner:        domainService.ActorID(),
				vocab.PropertyPublicKeyPEM: publicKeyPEM,
			},
		}

		return ctx.JSON(200, result)
	}
}
