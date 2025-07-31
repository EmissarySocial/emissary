package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

func GetPublicKey(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetPublicKey"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFoundError(location, "")
	}

	// Try to load the key from the Datbase
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()

	if err := keyService.LoadByParentID(session, model.EncryptionKeyTypeUser, user.UserID, &key); err != nil {
		return derp.Wrap(err, location, "Error loading encryption key for user", user.UserID)
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
