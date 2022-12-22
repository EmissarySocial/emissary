package handler

import (
	"fmt"
	"net/http"
	"strconv"

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

		fmt.Println("WebSub Client:::")
		fmt.Println("mode: " + transaction.Mode)
		fmt.Println("topic: " + transaction.Topic)
		fmt.Println("challenge: " + transaction.Challenge)
		fmt.Println("lease: " + strconv.FormatInt(transaction.Lease, 10))
		fmt.Println("userID: " + ctx.Param("userId"))
		fmt.Println("followingID: " + ctx.Param("followingId"))

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

		// RULE: Require that this Following uses WebSub
		if following.Method != model.FollowMethodWebSub {
			fmt.Println("!! Not a WebSub follow")
			return derp.New(derp.CodeBadRequestError, location, "Not a WebSub follow", following, transaction)
		}

		// RULE: Require that the Topic URL matches this Following
		if following.URL != transaction.Topic {
			fmt.Println("!! Invalid WebSub topic")
			return derp.New(derp.CodeBadRequestError, location, "Invalid WebSub topic", following, transaction)
		}

		// RULE: Force another poll in half the time of this lease
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

		followingService.Poll(&following)

		return nil
	}
}
