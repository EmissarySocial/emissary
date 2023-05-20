package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ActivityPub_GetBlockedCollection(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetBlocked"

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
			return derp.New(derp.CodeForbiddenError, location, "")
		}

		publishDateString := ctx.QueryParam("publishDate")

		// For requests directly to the collection, return a summary and the URL of the first page
		if publishDateString == "" {

			ctx.Response().Header().Set("Content-Type", "application/activity+json")
			result := activityPub_Collection(user.ActivityPubBlockedURL())
			return ctx.JSON(200, result)
		}

		// Fallthrough means this is a request for a specific page
		blockService := factory.Block()
		publishDate := convert.Int64(publishDateString)
		pageSize := 60
		blocks, err := blockService.QueryPublicBlocks(user.UserID, publishDate, option.MaxRows(int64(pageSize)))

		if err != nil {
			return derp.Wrap(err, location, "Error loading blocks")
		}

		// Return results to the client.
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		results := activityPub_CollectionPage(user.ActivityPubBlockedURL(), pageSize, blocks)
		return ctx.JSON(200, results)
	}
}

func ActivityPub_GetBlock(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Collect BlockID from URL
		blockID, err := primitive.ObjectIDFromHex(ctx.Param("block"))

		if err != nil {
			return derp.NewNotFoundError("handler.ActivityPub_GetLikedRecord", "Invalid Block ID", err)
		}

		// Validate the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.ActivityPub_GetLikedRecord", "Unrecognized domain name")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByToken(ctx.Param("user"), &user); err != nil {
			return derp.NewNotFoundError("handler.ActivityPub_GetLikedRecord", "User not found", err)
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.New(derp.CodeForbiddenError, "handler.ActivityPub_GetLikedRecord", "")
		}

		// Try to load the Block from the database
		blockService := factory.Block()
		block := model.NewBlock()

		if err := blockService.LoadByID(user.UserID, blockID, &block); err != nil {
			return derp.Wrap(err, "handler.ActivityPub_GetLikedRecord", "Error loading block")
		}

		// Return the block as JSON-LD
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusOK, block.GetJSONLD())
	}
}
