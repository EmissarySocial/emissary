package handler

import (
	"strconv"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/labstack/echo/v4"
)

func ActivityPub_GetBlocked(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_GetBlocked"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain name")
		}

		userService := factory.User()
		user := model.NewUser()
		userToken := ctx.Param("userId")

		if err := userService.LoadByToken(userToken, &user); err != nil {
			return derp.NewNotFoundError(location, "User not found", err)
		}

		collectionURL := user.ActivityPubBlockedURL()
		publishDateString := ctx.QueryParam("publishDate")

		// For requests directly to the collection, return a summary and the URL of the first page
		if publishDateString == "" {

			result := streams.NewOrderedCollection()
			result.Summary = "Block list published by " + user.Username
			result.First = collectionURL + "?publishDate=0"

			ctx.Response().Header().Set("Content-Type", "application/activity+json")
			return ctx.JSON(200, result)
		}

		// Fallthrough means this is a request for a specific page

		// Set up the response
		result := streams.NewOrderedCollectionPage()
		result.PartOf = collectionURL
		result.OrderedItems = make([]any, 0)

		blockService := factory.Block()
		publishDate := convert.Int64(publishDateString)
		pageSize := 2
		blocks, err := blockService.QueryPublicBlocks(user.UserID, publishDate, option.MaxRows(int64(pageSize)))

		if err != nil {
			return derp.Wrap(err, location, "Unable to load blocks")
		}

		// If there are results, then add them into the collection
		if len(blocks) > 0 {

			result.OrderedItems = slice.Map(blocks, func(block model.Block) any {
				return block.GetJSONLD()
			})

			lastBlock := blocks[len(blocks)-1]

			if len(blocks) >= pageSize {
				result.Next = collectionURL + "?publishDate=" + strconv.FormatInt(lastBlock.PublishDate, 10)
			}
		}

		// Return results to the client.
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(200, result)
	}
}
