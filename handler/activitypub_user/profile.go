package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

func RenderProfileJSONLD(context echo.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.RenderProfileJSONLD"

	// Try to load the key from the Datbase
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()

	if err := keyService.LoadByParentID(session, model.EncryptionKeyTypeUser, user.UserID, &key); err != nil {
		return derp.Wrap(err, location, "Unable to load encryption key for user", user.UserID)
	}

	// Combine the Profile and the EncryptionKey
	userJSON := user.GetJSONLD()
	userJSON[vocab.PropertyPublicKey] = mapof.Any{
		vocab.PropertyID:           user.ActivityPubPublicKeyURL(),
		vocab.PropertyOwner:        user.ActivityPubURL(),
		vocab.PropertyPublicKeyPEM: key.PublicPEM,
	}

	// Return the user's profile in JSON-LD format
	context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
	return context.JSON(http.StatusOK, userJSON)
}
