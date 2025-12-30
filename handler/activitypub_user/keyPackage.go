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

func GetKeyPackageCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetKeyPackageCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
	}

	// Fallthrough means this is a request for a specific page
	keyPackageService := factory.KeyPackage()
	keyPackages, err := keyPackageService.QueryIDOnlyByUser(session, user.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load rules")
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

func GetKeyPackageRecord(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetKeyPackageRecord"

	// Confirm that the user is visible
	if !isUserVisible(ctx, user) {
		return ctx.NoContent(http.StatusNotFound)
	}

	// Load the keyPackage from the database
	keyPackageService := factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByToken(session, user.UserID, ctx.Param("keyPackageId"), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Unable to load keyPackage")
	}

	result := keyPackageService.GetJSONLD(&keyPackage)

	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}
