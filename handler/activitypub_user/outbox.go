package activitypub_user

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
)

func GetOutboxCollection(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.activitypub.ActivityPub_GetOutboxCollection"

	return func(ctx echo.Context) error {

		// Validate the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain name")
		}

		// Try to load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByToken(ctx.Param("userId"), &user); err != nil {
			return derp.NewNotFoundError(location, "User not found", err)
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NewNotFoundError(location, "User not found")
		}

		// If the request is for the collection itself, then return a summary and the URL of the first page
		publishDateString := ctx.QueryParam("publishDate")

		if publishDateString == "" {
			ctx.Response().Header().Set("Content-Type", "application/activity+json")
			result := activitypub.Collection(user.ActivityPubOutboxURL())
			return ctx.JSON(http.StatusOK, result)
		}

		// Fall through means that we're looking for a specific page of the collection
		publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
		outboxService := factory.Outbox()
		pageID := fullURL(factory, ctx)
		pageSize := 60

		// Retrieve a page of messages from the database
		messages, err := outboxService.QueryByParentAndDate(model.FollowerTypeUser, user.UserID, publishedDate, pageSize)

		if err != nil {
			return derp.Wrap(err, location, "Error loading outbox messages")
		}

		// Return results as an OrderedCollectionPage
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.CollectionPage(pageID, user.ActivityPubOutboxURL(), pageSize, messages)
		return ctx.JSON(http.StatusOK, result)
	}
}
