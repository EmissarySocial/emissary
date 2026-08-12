package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
)

// GetKeyPackageCollection serves the User's published MLS KeyPackages as a collection of IDs.
// The route runs behind WithAuthorizedActorAndUser (R10): the requester is already identified
// and confirmed not-blocked before this handler runs.
func GetKeyPackageCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetKeyPackageCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
	}

	// RULE: Verify that the Domain allows MLS messages for this User
	if domain := factory.Domain().Get(); !domain.UserCanMLS(user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	// Fallthrough means this is a request for a specific page
	keyPackageService := factory.KeyPackage()
	keyPackages, err := keyPackageService.QueryIDOnlyByUser(session, user.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Loading rules")
	}

	collection := streams.NewCollection(user.ActivityPubKeyPackagesURL())
	collection.TotalItems = keyPackages.Length()
	collection.Items = slice.Map(keyPackages, func(item model.IDOnly) any {
		return keyPackageService.ActivityPubURL(user.UserID, item.ID)
	})

	// Return results to the client.
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(200, collection)
}

// GetKeyPackageRecord serves a single MLS KeyPackage. The route runs behind
// WithAuthorizedActorAndUser (R10): the requester is already identified and confirmed
// not-blocked before this handler runs.
func GetKeyPackageRecord(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetKeyPackageRecord"

	// Confirm that the user is visible
	if !isUserVisible(ctx, user) {
		return ctx.NoContent(http.StatusNotFound)
	}

	// RULE: Verify that the Domain allows MLS messages for this User
	if domain := factory.Domain().Get(); !domain.UserCanMLS(user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	// Load the keyPackage from the database
	keyPackageService := factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByToken(session, user.UserID, ctx.Param("keyPackageId"), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Loading keyPackage")
	}

	result := keyPackageService.GetJSONLD(&keyPackage)

	// Rewrite the generator for non-owners to only include the ID, not the name
	if authorization := getAuthorization(ctx); authorization.UserID != user.UserID {
		result["generator"] = result.GetMap("generator").GetString("id")
	}

	// Success
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}
