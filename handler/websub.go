package handler

import (
	"fmt"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
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

		fmt.Println("..following loaded successfully")

		/* RULE: Require that this Following uses WebSub
		if following.Method != model.FollowMethodWebSub {
			return derp.New(derp.CodeBadRequestError, location, "Not a WebSub follow", following, transaction)
		}*/

		// RULE: Update the Topic URL if it is not already set
		if self := following.GetLink("rel", "self"); !self.IsEmpty() {
			if transaction.Topic == self.Href {
				following.URL = self.Href
			}
		}

		// RULE: Require that the Topic URL matches this Following
		if transaction.Topic != following.URL {
			return derp.NewNotFoundError(location, "Invalid WebSub topic", following, transaction)
		}

		// RULE: Force another poll in half the time of this lease
		following.Method = model.FollowMethodWebSub
		following.PollDuration = int(transaction.Lease / 60 / 60 / 2) // poll again in half the lease duration

		// Update the record status and save.
		if err := followingService.SetStatus(&following, model.FollowingStatusSuccess, ""); err != nil {
			return derp.Wrap(err, "handler.getWebSubClient_subscribe", "Error updating following status", following)
		}

		fmt.Println("..following status updated successfully")

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
		if following.Method != model.FollowMethodWebSub {
			return derp.New(derp.CodeBadRequestError, location, "Not a WebSub follow", following)
		}

		// TODO: LOW: Validate the secret (HMAC how?)
		// TODO: LOW: Fat Pings require HMAC
		/*
			if following.Secret != "" {
				signature := ctx.Request().Header.Get("X-Hub-Signature")
			}
		*/

		followingService.Poll(&following)

		return nil
	}
}
