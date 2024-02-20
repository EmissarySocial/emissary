package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

func RenderProfileJSONLD(context echo.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.activitypub.renderProfileJSONLD"

	// Try to load the key from the Datbase
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()

	if err := keyService.LoadByID(user.UserID, &key); err != nil {
		return derp.Wrap(err, location, "Error loading encryption key for user", user.UserID)
	}

	// Combine the Profile and the EncryptionKey
	userJSON := user.GetJSONLD()
	userJSON[vocab.PropertyPublicKey] = mapof.Any{
		vocab.PropertyID:   user.ActivityPubURL() + "#main-key",
		vocab.PropertyType: "Key",
		"owner":            user.ActivityPubURL(),
		"publicKeyPem":     key.PublicPEM,
	}

	// Return the user's profile in JSON-LD format
	context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
	return context.JSON(http.StatusOK, userJSON)
}
