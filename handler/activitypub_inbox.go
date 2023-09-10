package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ActivityPub_PostInbox(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.ActivityPub_PostInbox"

	return func(ctx echo.Context) error {

		// Find the factory for this hostname
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error creating ActivityStreamsHandler")
		}

		// Try to load the User who owns this inbox
		userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

		if err != nil {
			return derp.Wrap(err, location, "UserID must be a valid ObjectID")
		}

		user := model.NewUser()
		userService := factory.User()
		if err := userService.LoadByID(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", userID.Hex())
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NewNotFoundError(location, "")
		}

		// Retrieve the activity from the request body
		activity, err := pub.ReceiveInboxRequest(ctx.Request(), factory.ActivityStreams())

		if err != nil {
			err = derp.Wrap(err, location, "Error parsing ActivityPub request")
			derp.Report(err)
			return err
		}

		// Handle the ActivityPub request
		if err := inboxRouter.Handle(factory, &user, activity); err != nil {
			err := derp.Wrap(err, location, "Error handling ActivityPub request")
			derp.Report(err)
			return err
		}

		// Send the response to the client
		return ctx.String(http.StatusOK, "")
	}
}
