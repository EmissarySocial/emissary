package handler

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type webSubConfirmation struct {
	Mode      string `query:"hub.mode"`
	Topic     string `query:"hub.topic"`
	Challenge string `query:"hub.challenge"`
	Lease     int64  `query:"hub.lease_seconds"`
}

func GetWebSubClient(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		const location = "handler.GetWebSubClient"

		var transaction webSubConfirmation

		// Collect the URL variables
		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Error parsing WebSub transaction", ctx.Request().URL)
		}

		// If this is not a subscription confirmation (i.e. a delete confirmation), then we're done.
		if transaction.Mode != "subscribe" {
			return nil
		}

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading server factory")
		}

		// Parse the UserID from the query string
		userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid UserID", userID)
		}

		// Parse the Following from the query string
		followingID, err := primitive.ObjectIDFromHex(ctx.Param("followingId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid FollowingID", followingID)
		}

		// Load the following record from the database
		followingService := factory.Following()
		following := model.NewFollowing()

		if err := followingService.LoadByID(userID, followingID, &following); err != nil {
			return derp.Wrap(err, location, "Error loading following record", userID, followingID, transaction)
		}

		// Validate the request (B)
		if following.UpdateMethod != model.FollowUpdateMethodWebSub {
			return derp.New(derp.CodeBadRequestError, location, "Not a WebSub follow", following, transaction)
		}

		if following.ResourceURL != transaction.Topic {
			return derp.New(derp.CodeBadRequestError, location, "Invalid WebSub topic", following, transaction)
		}

		following.PollDuration = int(transaction.Lease / 60 / 60 / 2) // poll again in half the lease duration
		following.Expiration = time.Now().Add(time.Duration(transaction.Lease) * time.Second).Unix()

		// Update the record status and save.
		if err := followingService.SetStatus(&following, model.FollowingStatusSuccess, ""); err != nil {
			return derp.Wrap(err, "handler.getWebSubClient_subscribe", "Error updating following status", following)
		}

		// Win!
		return ctx.String(http.StatusOK, transaction.Challenge)
	}
}

func PostWebSubClient(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		const location = "handler.GetWebSubClient"

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading server factory")
		}

		// Parse the UserID from the query string
		userID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid UserID", userID)
		}

		// Parse the Following from the query string
		followingID, err := primitive.ObjectIDFromHex(ctx.Param("followingId"))

		if err != nil {
			return derp.Wrap(err, location, "Invalid FollowingID", followingID)
		}

		// Load the following record from the database
		followingService := factory.Following()
		following := model.NewFollowing()

		if err := followingService.LoadByID(userID, followingID, &following); err != nil {
			return derp.Wrap(err, location, "Error loading following record", userID, followingID)
		}

		// Validate the request (B)
		if following.UpdateMethod != model.FollowUpdateMethodWebSub {
			return derp.New(derp.CodeBadRequestError, location, "Not a WebSub follow", following)
		}

		// TODO: MEDIUM: Validate the secret (HMAC how?)
		spew.Dump(following.Secret)

		followingService.Poll(&following)

		return nil
	}
}
