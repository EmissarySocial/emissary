// Package activitypub_user serves the ActivityPub endpoints for a User actor: the actor document
// itself, its inbox and outbox, and its social-graph collections.
//
// The followers and following collections publish only their SIZE, never their members -- see
// GetFollowersCollection for the policy and the reasoning behind that response shape.
package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// RenderProfileJSONLD serves the User's complete ActivityPub actor document as JSON-LD
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
	headers.SetAll(context.Response().Header(), headers.VariantActivityPub, user)
	return context.JSON(http.StatusOK, userJSON)
}
