package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// pushSubscriptionRequest is the JSON body posted by the browser.  It mirrors the shape that
// PushManager.subscribe() returns: {endpoint, keys:{p256dh, auth}}.
type pushSubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// PostPushSubscription upserts a Web Push subscription for the authenticated User
func PostPushSubscription(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.PostPushSubscription"

	// Parse the browser's subscription from the request body
	var body pushSubscriptionRequest

	if err := ctx.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Parsing request body", derp.WithBadRequest())
	}

	// RULE: A subscription is useless without its endpoint and both crypto keys
	if body.Endpoint == "" || body.Keys.P256DH == "" || body.Keys.Auth == "" {
		return derp.BadRequest(location, "endpoint, keys.p256dh, and keys.auth are all required")
	}

	// RULE: Refuse endpoints that point at an internal address, so the server cannot be used to
	// probe or reach hosts behind its firewall (SSRF).  Delivery applies the same guard at dial time.
	if !factory.WebPush().EndpointIsAllowed(body.Endpoint) {
		return derp.Forbidden(location, "Push endpoint is not allowed")
	}

	// Bind the subscription to this User.  The userID comes from the session, never the body, so a
	// caller cannot name a different owner; Upsert separately refuses another User's endpoint.
	userAgent := ctx.Request().UserAgent()

	if err := factory.PushSubscription().Upsert(session, user.UserID, body.Endpoint, body.Keys.P256DH, body.Keys.Auth, userAgent); err != nil {
		return derp.Wrap(err, location, "Saving PushSubscription")
	}

	return ctx.NoContent(http.StatusNoContent)
}

// DeletePushSubscription removes a Web Push subscription (on explicit unsubscribe)
func DeletePushSubscription(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.DeletePushSubscription"

	// Parse the endpoint to unsubscribe from the request body
	var body pushSubscriptionRequest

	if err := ctx.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Parsing request body", derp.WithBadRequest())
	}

	// RULE: The endpoint names the subscription to remove, so it is required
	if body.Endpoint == "" {
		return derp.BadRequest(location, "endpoint is required")
	}

	// Load the subscription that claims this endpoint
	subscription := model.NewPushSubscription()

	if err := factory.PushSubscription().LoadByEndpoint(session, body.Endpoint, &subscription); err != nil {

		// A subscription that is already gone is an idempotent success, not a failure
		if derp.IsNotFound(err) {
			return ctx.NoContent(http.StatusNoContent)
		}

		return derp.Wrap(err, location, "Loading PushSubscription")
	}

	// RULE: Only the owner may unsubscribe their own device
	if subscription.UserID != user.UserID {
		return derp.Forbidden(location, "You do not have permission to delete this subscription")
	}

	// Remove the subscription
	if err := factory.PushSubscription().DeleteByEndpoint(session, body.Endpoint, "unsubscribe"); err != nil {
		return derp.Wrap(err, location, "Deleting PushSubscription")
	}

	return ctx.NoContent(http.StatusNoContent)
}
