package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
)

func GetResponseCollection(ctx *steranko.Context, factory *domain.Factory, session data.Session, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetResponseCollection"

	// RULE: Only PUBLIC streams have /likes /dislikes and /mentions
	if !stream.DefaultAllowAnonymous() {
		return derp.UnauthorizedError(location, "Anonymous access not allowed")
	}

	// Parse the Response Type from the URL
	responseType := getResponseType(ctx)

	// If the request is for the collection itself, then return a summary and the URL of the first page
	publishDateString := ctx.QueryParam("publishDate")
	baseRequestURL := stream.ActivityPubResponses(responseType)

	// If no "publishedDate" then return the collection header.
	if publishDateString == "" {
		if publishDateString == "" {
			ctx.Response().Header().Set("Content-Type", "application/activity+json")
			result := activitypub.Collection(baseRequestURL)
			return ctx.JSON(http.StatusOK, result)
		}
	}

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64(publishDateString)
	responseService := factory.Response()
	pageID := fullURL(factory, ctx)
	pageSize := 60

	// Retrieve a page of responses from the database
	responses, err := responseService.QueryByObjectAndDate(session, stream.Permalink(), responseType, publishedDate, pageSize)

	if err != nil {
		return derp.Wrap(err, location, "Error loading responses")
	}

	// Return a JSON-LD document
	ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)

	result := activitypub.CollectionPage(
		pageID,
		baseRequestURL,
		pageSize,
		responses,
	)

	return ctx.JSON(http.StatusOK, result)
}
