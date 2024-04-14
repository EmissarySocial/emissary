package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostFollowEmail(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostFollowEmail"

	return func(ctx echo.Context) error {

		transaction := struct {
			ParentID primitive.ObjectID `form:"parentId"`
			Type     string             `form:"type"`
			Name     string             `form:"name"`
			Email    string             `form:"email"`
		}{}

		// Collect inputs from the context
		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Unable to bind input")
		}

		// Get the domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error locating domain")
		}

		// Create the new "Follower" record.
		follower := model.NewFollower()
		follower.StateID = model.FollowerStatePending
		follower.Type = transaction.Type
		follower.ParentID = transaction.ParentID
		follower.Method = model.FollowerMethodEmail
		follower.Format = model.MimeTypeHTML
		follower.Actor.EmailAddress = transaction.Email
		follower.Actor.Name = transaction.Name

		// Save the follower
		followerService := factory.Follower()
		if err := followerService.Save(&follower, "Added follower via email"); err != nil {
			return derp.Wrap(err, location, "Error saving follower")
		}

		// Forward the user to the confirmation page
		return ctx.Redirect(http.StatusFound, "/@"+follower.ParentID.Hex()+"/follow-email-sent")
	}
}
