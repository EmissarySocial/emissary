package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetResponseCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.GetResponseCollection"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
	}

	// Parse the Response Type from the URL
	responseType := getResponseType(ctx)

	if responseType == "" {
		return derp.NotFound(location, "Invalid response type", "Valid types are: 'shared', 'liked', 'disliked'")
	}

	// If the request is for the collection itself, then return a summary and the URL of the first page
	publishDateString := ctx.QueryParam("publishDate")

	if publishDateString == "" {
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.Collection(user.ActivityPubLikedURL())
		return ctx.JSON(http.StatusOK, result)
	}

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64(publishDateString)
	responseService := factory.Response()
	pageID := fullURL(factory, ctx)
	pageSize := 60

	// Retrieve a page of responses from the database
	responses, err := responseService.QueryByUserAndDate(session, user.UserID, responseType, publishedDate, pageSize)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load responses")
	}

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := activitypub.CollectionPage(pageID, user.ActivityPubLikedURL(), pageSize, responses)
	return ctx.JSON(http.StatusOK, result)
}

func GetResponse(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub.ActivityPub_GetUserResponse"

	// Collect ResponseID from URL
	responseID, err := primitive.ObjectIDFromHex(ctx.Param("response"))

	if err != nil {
		return derp.NotFound(location, "Invalid Response ID", err)
	}

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFound(location, "User not found")
	}

	// Try to load the Response from the database
	responseService := factory.Response()
	response := model.NewResponse()

	if err := responseService.LoadByID(session, user.UserID, responseID, &response); err != nil {
		return derp.Wrap(err, location, "Unable to load response")
	}

	if response.Actor != user.ProfileURL {
		return derp.NotFound(location, "Response not found", "ActorID does not match")
	}

	// Return the response as JSON-LD
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, response.GetJSONLD())
}
