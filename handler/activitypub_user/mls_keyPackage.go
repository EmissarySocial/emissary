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

// GetKeyPackageCollection serves the User's published MLS KeyPackages as a collection of IDs
func GetKeyPackageCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetKeyPackageCollection"

	// This route is PUBLIC -- server.go registers it behind handler.WithUser -- because
	// KeyPackages are published material that a peer must fetch before any authenticated
	// relationship exists. The requester is neither identified nor checked against the User's
	// Rules, so the two RULEs below are the only gate. (BUG-21)

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
	}

	// RULE: Verify that the Domain allows MLS messages for this User
	if domain := factory.Domain().Get(); !domain.UserCanMLS(user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	// Load every KeyPackage this User has published
	keyPackageService := factory.KeyPackage()
	keyPackages, err := keyPackageService.QueryIDOnlyByUser(session, user.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Loading KeyPackages")
	}

	// Map the records into a Collection of ActivityPub URLs
	collection := streams.NewCollection(user.ActivityPubKeyPackagesURL())
	collection.TotalItems = keyPackages.Length()
	collection.Items = slice.Map(keyPackages, func(item model.IDOnly) any {
		return keyPackageService.ActivityPubURL(user.UserID, item.ID)
	})

	// Return results to the client.
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(200, collection)
}

// GetKeyPackageRecord serves a single MLS KeyPackage, named by its token in the request URL
func GetKeyPackageRecord(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetKeyPackageRecord"

	// This route is PUBLIC -- server.go registers it behind handler.WithUser -- because a peer
	// must fetch a KeyPackage before any authenticated relationship exists. Requester identity
	// therefore comes from the steranko session, NOT from an ActivityPub signature: an anonymous
	// requester resolves to the zero UserID, so the owner check below takes the non-owner path,
	// which is the safe default. (BUG-21)

	// RULE: Only visible users can be queried
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

	// Serialize the KeyPackage into its JSON-LD representation
	result := keyPackageService.GetJSONLD(&keyPackage)

	// RULE: Non-owners see only the generator's ID, never its name
	if authorization := getAuthorization(ctx); authorization.UserID != user.UserID {
		result["generator"] = result.GetMap("generator").GetString("id")
	}

	// Success
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}
