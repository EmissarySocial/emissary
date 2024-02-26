package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

func GetJSONLD(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "activitypub_stream.GetJSONLD"

	return func(ctx echo.Context) error {

		// Load all of the necessary object from the request
		factory, _, streamService, _, stream, actor, err := getActor(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Request Not Accepted")
		}

		// If this Stream is not an Actor, then just return a standard JSON-LD response.
		if actor.IsNil() {
			jsonld := streamService.JSONLD(&stream)
			ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
			return ctx.JSON(http.StatusOK, jsonld)
		}

		// Try to locate the domain
		// Try to load the Encryption Key for this Actor
		keyService := factory.EncryptionKey()
		key := model.NewEncryptionKey()
		if err := keyService.LoadByParentID(model.EncryptionKeyTypeStream, stream.StreamID, &key); err != nil {
			return derp.Wrap(err, location, "Error loading Public Key", stream.StreamID)
		}

		// Combine the Actor and the Public Key
		result := actor.JSONLD(&stream)
		result[vocab.PropertyPublicKey] = mapof.Any{
			vocab.PropertyID:   stream.Permalink() + "#main-key",
			vocab.PropertyType: "Key",
			"owner":            stream.Permalink(),
			"publicKeyPem":     key.PublicPEM,
		}

		// Return an ActivityPub response
		ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
		return ctx.JSON(http.StatusOK, result)
	}
}
