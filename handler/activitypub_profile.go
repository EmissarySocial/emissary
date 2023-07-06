package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

func renderProfileJSONLD(context echo.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.renderProfileJSONLD"

	// Try to load the key from the Datbase
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()

	if err := keyService.LoadByID(user.UserID, &key); err != nil {
		return derp.Wrap(err, location, "Error loading encryption key for user", user.UserID)
	}

	// Return the key as JSON-LD
	keyJSON := mapof.Any{
		"id":           user.ActivityPubURL() + "#main-key",
		"type":         "Key",
		"owner":        user.ActivityPubURL(),
		"publicKeyPem": key.PublicPEM,
	}

	userJSON := user.GetJSONLD()
	userJSON["publicKey"] = keyJSON

	// Return the user's profile in JSON-LD format
	context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
	return context.JSON(http.StatusOK, userJSON)
}
