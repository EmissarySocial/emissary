package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/inbox"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostInbox(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.activitypub.ActivityPub_PostInbox"

	return func(ctx echo.Context) error {

		// Find the factory for this hostname
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Domain")
		}

		// Try to load the User who owns this inbox
		userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

		if err != nil {
			return derp.Wrap(err, location, "UserID must be a valid ObjectID")
		}

		user := model.NewUser()
		userService := factory.User()
		if err := userService.LoadByID(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading User", userID.Hex(), derp.WithCode(http.StatusGone))
		}

		// RULE: Only public users can be queried
		if !user.IsPublic {
			return derp.NewNotFoundError(location, "")
		}

		// Retrieve the activity from the request body
		activity, err := inbox.ReceiveRequest(ctx.Request(), factory.ActivityStream())

		if err != nil {
			return derp.Wrap(err, location, "Error parsing ActivityPub request")
		}

		// Create a new Context
		context := Context{
			factory: factory,
			user:    &user,
		}

		// Handle the ActivityPub request
		if err := inboxRouter.Handle(context, activity); err != nil {
			return derp.Wrap(err, location, "Error handling ActivityPub request")
		}

		// Send the response to the client
		return ctx.String(http.StatusOK, "")
	}
}
