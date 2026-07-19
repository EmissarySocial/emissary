package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

func RenderProfileJSONLD(context *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.RenderProfileJSONLD"

	// RULE: Non-public profiles are hidden from anonymous/remote requesters, matching the HTML
	// path and every sibling ActivityPub endpoint. The domain owner and the user themselves are allowed.
	if !isUserVisible(context, user) {
		return derp.NotFound(location, "User not found")
	}

	// Try to load the key from the Datbase
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()

	if err := keyService.LoadByParentID(session, model.EncryptionKeyTypeUser, user.UserID, &key); err != nil {
		return derp.Wrap(err, location, "Loading encryption key for user", user.UserID)
	}

	// Combine the Profile and the EncryptionKey
	userJSON := user.GetJSONLD()
	userJSON[vocab.PropertyPublicKey] = mapof.Any{
		vocab.PropertyID:           user.ActivityPubPublicKeyURL(),
		vocab.PropertyOwner:        user.ActivityPubURL(),
		vocab.PropertyPublicKeyPEM: key.PublicPEM,
	}

	// If the domain allows it, append MLS messaging values as well.
	domainService := factory.Domain()
	domain := domainService.Get()

	if domain.UserCanMLS(user) {
		userJSON[vocab.PropertyMLSMessages] = user.ActivityPubInboxURL_DirectMessages_MLS()
		userJSON[vocab.PropertyMLSKeyPackages] = user.ActivityPubKeyPackagesURL()
	}

	// Return the user's profile in JSON-LD format
	context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
	return context.JSON(http.StatusOK, userJSON)
}
