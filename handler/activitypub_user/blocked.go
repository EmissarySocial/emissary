package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetBlockedCollection(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.activitypub_user.GetBlockedCollection"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain name")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()
		userToken := ctx.Param("userId")

		if err := userService.LoadByToken(userToken, &user); err != nil {
			return derp.NewNotFoundError(location, "User not found", err)
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NewNotFoundError(location, "User not found")
		}

		publishDateString := ctx.QueryParam("publishDate")

		// For requests directly to the collection, return a summary and the URL of the first page
		if publishDateString == "" {

			ctx.Response().Header().Set("Content-Type", "application/activity+json")
			result := activitypub.Collection(user.ActivityPubBlockedURL())
			return ctx.JSON(200, result)
		}

		// Fallthrough means this is a request for a specific page
		ruleService := factory.Rule()
		publishDate := convert.Int64(publishDateString)
		pageID := fullURL(factory, ctx)
		pageSize := 60
		rules, err := ruleService.QueryPublic(user.UserID, publishDate, option.MaxRows(int64(pageSize)))

		if err != nil {
			return derp.Wrap(err, location, "Error loading rules")
		}

		// Convert the slice of rules into JSONLDGetters
		jsonldGetters := slice.Map(rules, func(rule model.Rule) service.RuleJSONLDGetter {
			return ruleService.JSONLDGetter(rule)
		})

		// Return results to the client.
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		results := activitypub.CollectionPage(pageID, user.ActivityPubBlockedURL(), pageSize, jsonldGetters)
		return ctx.JSON(200, results)
	}
}

func GetBlock(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.activitypub.ActivityPub_GetBlock"

	return func(ctx echo.Context) error {

		// Collect RuleID from URL
		ruleID, err := primitive.ObjectIDFromHex(ctx.Param("ruleId"))

		if err != nil {
			return derp.NewNotFoundError(location, "Invalid Rule ID", err)
		}

		// Validate the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain name")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByToken(ctx.Param("userId"), &user); err != nil {
			return derp.NewNotFoundError(location, "User not found", err)
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NewNotFoundError(location, "User not found")
		}

		// Try to load the Rule from the database
		ruleService := factory.Rule()
		rule := model.NewRule()

		if err := ruleService.LoadByID(user.UserID, ruleID, &rule); err != nil {
			return derp.Wrap(err, location, "Error loading rule")
		}

		// Return the rule as JSON-LD
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusOK, ruleService.JSONLD(rule))
	}
}
