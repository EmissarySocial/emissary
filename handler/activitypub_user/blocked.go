package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetBlockedCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetBlockedCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
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
	rules, err := ruleService.QueryPublic(session, user.UserID, publishDate, option.MaxRows(int64(pageSize)))

	if err != nil {
		return derp.Wrap(err, location, "Unable to load rules")
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

func GetBlock(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub.ActivityPub_GetBlock"

	// Collect RuleID from URL
	ruleID, err := primitive.ObjectIDFromHex(ctx.Param("ruleId"))

	if err != nil {
		return derp.NotFound(location, "Invalid Rule ID", err)
	}

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
	}

	// Try to load the Rule from the database
	ruleService := factory.Rule()
	rule := model.NewRule()

	if err := ruleService.LoadByID(session, user.UserID, ruleID, &rule); err != nil {
		return derp.Wrap(err, location, "Unable to load rule")
	}

	// Return the rule as JSON-LD
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, ruleService.JSONLD(rule))
}
