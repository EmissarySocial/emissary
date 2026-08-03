package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

func RenderProfileJSONLD(context *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.RenderProfileJSONLD"

	// RULE: Non-public profiles are hidden from anonymous/remote requesters, matching the HTML
	// path and every sibling ActivityPub endpoint. The domain owner and the user themselves are allowed.
	if !isUserVisible(context, user) {
		return derp.NotFound(location, "User not found")
	}

	// Assemble the complete actor document (profile + publicKey + MLS). This is the same
	// assembly used for outbound profile Updates, so the two representations cannot drift.
	userJSON, err := factory.User().ActivityPubProfile(session, user)

	if err != nil {
		return derp.Wrap(err, location, "Assembling actor document", user.UserID)
	}

	// Return the user's profile in JSON-LD format
	context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
	return context.JSON(http.StatusOK, userJSON)
}
