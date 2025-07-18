package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetResponseCollection(serverFactory *server.Factory, responseType string) echo.HandlerFunc {

	const location = "handler.activitypub_user.GetResponseCollection"

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
			return derp.NotFoundError(location, "User not found", err)
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NotFoundError(location, "User not found")
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
		responses, err := responseService.QueryByUserAndDate(user.UserID, responseType, publishedDate, pageSize)

		if err != nil {
			return derp.Wrap(err, location, "Error loading responses")
		}

		// Return results as an OrderedCollectionPage
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		result := activitypub.CollectionPage(pageID, user.ActivityPubLikedURL(), pageSize, responses)
		return ctx.JSON(http.StatusOK, result)
	}
}

func GetResponse(serverFactory *server.Factory, responseType string) echo.HandlerFunc {

	const location = "handler.activitypub.ActivityPub_GetUserResponse"

	return func(ctx echo.Context) error {

		// Collect ResponseID from URL
		responseID, err := primitive.ObjectIDFromHex(ctx.Param("response"))

		if err != nil {
			return derp.NotFoundError(location, "Invalid Response ID", err)
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
			return derp.NotFoundError(location, "User not found", err)
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NotFoundError(location, "User not found")
		}

		// Try to load the Response from the database
		responseService := factory.Response()
		response := model.NewResponse()

		if err := responseService.LoadByID(responseID, &response); err != nil {
			return derp.Wrap(err, location, "Error loading response")
		}

		if response.Actor != user.ProfileURL {
			return derp.NotFoundError(location, "Response not found", "ActorID does not match")
		}

		if response.Type != responseType {
			return derp.NotFoundError(location, "Response not found", "Type does not match")
		}

		// Return the response as JSON-LD
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusOK, response.GetJSONLD())
	}
}
