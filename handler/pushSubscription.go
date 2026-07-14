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

// PostPushSubscription upserts a Web Push subscription for the authenticated User.  The userID comes
// from the session (never the request body), so one User cannot register a subscription for another.
func PostPushSubscription(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.PostPushSubscription"

	var body pushSubscriptionRequest

	if err := ctx.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Unable to parse request body", derp.WithBadRequest())
	}

	if body.Endpoint == "" || body.Keys.P256DH == "" || body.Keys.Auth == "" {
		return derp.BadRequest(location, "endpoint, keys.p256dh, and keys.auth are all required")
	}

	// RULE: Refuse endpoints that point at an internal address, so the server cannot be used to
	// probe or reach hosts behind its firewall (SSRF).  Delivery applies the same guard at dial time.
	if !factory.WebPush().EndpointIsAllowed(body.Endpoint) {
		return derp.Forbidden(location, "Push endpoint is not allowed")
	}

	userAgent := ctx.Request().UserAgent()

	if err := factory.PushSubscription().Upsert(session, user.UserID, body.Endpoint, body.Keys.P256DH, body.Keys.Auth, userAgent); err != nil {
		return derp.Wrap(err, location, "Unable to save push subscription")
	}

	return ctx.NoContent(http.StatusNoContent)
}

// DeletePushSubscription removes a Web Push subscription (on explicit unsubscribe).  It only deletes
// a subscription that belongs to the authenticated User.
func DeletePushSubscription(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.DeletePushSubscription"

	var body pushSubscriptionRequest

	if err := ctx.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Unable to parse request body", derp.WithBadRequest())
	}

	if body.Endpoint == "" {
		return derp.BadRequest(location, "endpoint is required")
	}

	// Verify the subscription belongs to this user before deleting.
	subscription := model.NewPushSubscription()
	err := factory.PushSubscription().LoadByEndpoint(session, body.Endpoint, &subscription)

	if derp.IsNotFound(err) {
		return ctx.NoContent(http.StatusNoContent) // Already gone — idempotent success.
	}

	if err != nil {
		return derp.Wrap(err, location, "Unable to load push subscription")
	}

	if subscription.UserID != user.UserID {
		return derp.Forbidden(location, "You do not have permission to delete this subscription")
	}

	if err := factory.PushSubscription().DeleteByEndpoint(session, body.Endpoint, "unsubscribe"); err != nil {
		return derp.Wrap(err, location, "Unable to delete push subscription")
	}

	return ctx.NoContent(http.StatusNoContent)
}
