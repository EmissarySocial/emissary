package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
)

func ActivityPub_GetStreamLikes(serverFactory *server.Factory) echo.HandlerFunc {
	return activityPub_getStreamResponseCollection(serverFactory, model.ResponseTypeLike)
}

func ActivityPub_GetStreamDislikes(serverFactory *server.Factory) echo.HandlerFunc {
	return activityPub_getStreamResponseCollection(serverFactory, model.ResponseTypeDislike)
}

func ActivityPub_GetStreamMentions(serverFactory *server.Factory) echo.HandlerFunc {
	return activityPub_getStreamResponseCollection(serverFactory, model.ResponseTypeMention)
}

func activityPub_getStreamResponseCollection(serverFactory *server.Factory, responseType string) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		const location = "handler.handleResponseCollection"

		// Verify the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error creating factory")
		}

		// Load the stream to verify the URL
		streamService := factory.Stream()
		stream := model.NewStream()
		token := ctx.Param("stream")

		if err := streamService.LoadByToken(token, &stream); err != nil {
			return derp.Wrap(err, location, "Error loading stream")
		}

		// RULE: Only PUBLIC streams have /likes /dislikes and /mentions
		if !stream.DefaultAllowAnonymous() {
			return derp.NewUnauthorizedError(location, "Anonymous access not allowed")
		}

		// If the request is for the collection itself, then return a summary and the URL of the first page
		publishDateString := ctx.QueryParam("publishDate")
		baseRequestURL := stream.ActivityPubResponses(responseType)

		// If no "publishedDate" then return the collection header.
		if publishDateString == "" {
			if publishDateString == "" {
				ctx.Response().Header().Set("Content-Type", "application/activity+json")
				result := activityPub_Collection(baseRequestURL)
				return ctx.JSON(http.StatusOK, result)
			}
		}

		// Fall through means that we're looking for a specific page of the collection
		publishedDate := convert.Int64(publishDateString)
		responseService := factory.Response()
		pageSize := 60

		// Retrieve a page of responses from the database
		responses, err := responseService.QueryByObjectAndDate(stream.Permalink(), responseType, publishedDate, pageSize)

		if err != nil {
			return derp.Wrap(err, location, "Error loading responses")
		}

		// Return a JSON-LD document
		ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)

		result := activityPub_CollectionPage[model.Response](
			baseRequestURL,
			pageSize,
			responses,
		)

		return ctx.JSON(http.StatusOK, result)
	}
}
