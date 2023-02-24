package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ActivityPub_GetPublicKey(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetPublicKey"

	return func(ctx echo.Context) error {

		// Parse the UserID parameter
		userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

		if err != nil {
			return derp.Wrap(err, location, "UserID must be a valid ObjectID", ctx.Param("userId"))
		}

		// Try to get the factory for this Domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error getting server factory")
		}

		// Try to load the key from the Datbase
		keyService := factory.EncryptionKey()
		key := model.NewEncryptionKey()

		if err := keyService.LoadByID(userID, &key); err != nil {
			return derp.Wrap(err, location, "Error loading encryption key for user", userID)
		}

		// Return the key as JSON-LD
		result := mapof.Any{
			"@context":     "https://w3id.org/security/v1",
			"id":           keyService.KeyID(&key),
			"owner":        keyService.OwnerID(&key),
			"publicKeyPem": key.PublicPEM,
		}

		return ctx.JSON(http.StatusOK, result)
	}
}
