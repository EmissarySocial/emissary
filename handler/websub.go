package handler

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/hmac"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type webSubConfirmation struct {
	Mode      string `query:"hub.mode"`
	Topic     string `query:"hub.topic"`
	Challenge string `query:"hub.challenge"`
	Lease     int64  `query:"hub.lease_seconds"`
}

// GetWebSubClient is called by an external WebSub server to confirm a subscription request.
func GetWebSubClient(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

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

	if err := followingService.LoadByID(session, userID, followingID, &following); err != nil {
		return derp.Wrap(err, location, "Unable to load following record", userID, followingID, transaction)
	}

	// RULE: Require that this Following uses WebSub
	if following.Method != model.FollowingMethodWebSub {
		return derp.BadRequestError(location, "Not a WebSub follow", following, transaction)
	}

	// RULE: Require that the Topic URL matches this Following
	if transaction.Topic != following.URL {
		return derp.NotFoundError(location, "Invalid WebSub topic", following, transaction)
	}

	// RULE: Force another poll in half the time of this lease
	following.Method = model.FollowingMethodWebSub
	following.PollDuration = int(transaction.Lease / 60 / 60 / 2) // poll again in half the lease duration

	// Update the record status and save.
	if err := followingService.SetStatusSuccess(session, &following); err != nil {
		return derp.Wrap(err, "handler.getWebSubClient_subscribe", "Unable to update following status", following)
	}

	// Win!
	return ctx.String(http.StatusOK, transaction.Challenge)

}

// PostWebSubClient is called by an external WebSub server to notify us of a change.
func PostWebSubClient(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetWebSubClient"

	var body bytes.Buffer

	if err := ctx.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Unable to read request body")
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

	if err := followingService.LoadByID(session, userID, followingID, &following); err != nil {
		return derp.Wrap(err, location, "Unable to load following record", userID, followingID)
	}

	// Validate the request (B)
	if following.Method != model.FollowingMethodWebSub {
		return derp.BadRequestError(location, "Not a WebSub follow", following)
	}

	// Validate the HMAC signature
	if following.Secret != "" {
		header := ctx.Request().Header.Get("X-Hub-Signature")
		method, signature := list.Equal(header).Split()

		hmac.Validate(method, following.Secret, body.Bytes(), signature.Bytes())
	}

	// TODO: MEDIUM: WebSub - Handle FatPings.
	// Right now, we re-poll the entire feed. But we could save a round-trip by
	// inspecting the body and parsing any additional data that's been "Fat Ping-ed" to us.

	// Connect to the the WebSub server
	if err := followingService.Connect(session, &following); err != nil {
		return derp.Wrap(err, location, "Error connecting to following", following)
	}

	// Woot woot!
	return nil
}
