package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/push/#create
func PostPushSubscription(serverFactory *server.Factory) func(model.Authorization, txn.PostPushSubscription) (object.WebPushSubscription, error) {

	const location = "handler.mastodon.PostPushSubscription"

	return func(auth model.Authorization, t txn.PostPushSubscription) (object.WebPushSubscription, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		endpoint := t.Subscription.Endpoint

		if err := factory.PushSubscription().ReplaceForOAuthUserToken(session, auth.UserID, auth.OAuthUserTokenID, endpoint, t.Subscription.Keys.P256dh, t.Subscription.Keys.Auth); err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Saving subscription")
		}

		// RULE: Emissary has nowhere to persist per-type alert preferences, so every notification
		// that clears the recipient's rules generates a push regardless. Echo the requested
		// alerts back rather than silently ignore them.
		return pushSubscriptionToot(factory, session, endpoint, requestedAlerts(t)), nil
	}
}

// https://docs.joinmastodon.org/methods/push/#get
func GetPushSubscription(serverFactory *server.Factory) func(model.Authorization, txn.GetPushSubscription) (object.WebPushSubscription, error) {

	const location = "handler.mastodon.GetPushSubscription"

	return func(auth model.Authorization, t txn.GetPushSubscription) (object.WebPushSubscription, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		sub := model.NewPushSubscription()

		if err := factory.PushSubscription().LoadByOAuthUserToken(session, auth.UserID, auth.OAuthUserTokenID, &sub); err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Loading subscription")
		}

		// RULE: alerts aren't stored (see PostPushSubscription); reporting them all enabled
		// matches what actually happens, since nothing filters push delivery today.
		return pushSubscriptionToot(factory, session, sub.Endpoint, allAlertsEnabled()), nil
	}
}

// https://docs.joinmastodon.org/methods/push/#update
func PutPushSubscription(serverFactory *server.Factory) func(model.Authorization, txn.PutPushSubscription) (object.WebPushSubscription, error) {

	const location = "handler.mastodon.PutPushSubscription"

	return func(auth model.Authorization, t txn.PutPushSubscription) (object.WebPushSubscription, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		sub := model.NewPushSubscription()

		if err := factory.PushSubscription().LoadByOAuthUserToken(session, auth.UserID, auth.OAuthUserTokenID, &sub); err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Loading subscription")
		}

		// RULE: alerts aren't stored (see PostPushSubscription), so echo the requested ones back
		return pushSubscriptionToot(factory, session, sub.Endpoint, updatedAlerts(t)), nil
	}
}

// DeletePushSubscription removes the calling app's push subscription
// https://docs.joinmastodon.org/methods/push/#delete
func DeletePushSubscription(serverFactory *server.Factory) func(model.Authorization, txn.DeletePushSubscription) (struct{}, error) {

	// RULE: As in Mastodon, this removes only the calling app's subscription, never the User's others.

	const location = "handler.mastodon.DeletePushSubscription"

	return func(auth model.Authorization, t txn.DeletePushSubscription) (struct{}, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		if err := factory.PushSubscription().DeleteByOAuthUserToken(session, auth.UserID, auth.OAuthUserTokenID, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Deleting subscription")
		}

		return struct{}{}, nil
	}
}

// pushSubscriptionToot builds the Mastodon API response for one PushSubscription.
func pushSubscriptionToot(factory *service.Factory, session data.Session, endpoint string, alerts object.WebPushSubscriptionAlerts) object.WebPushSubscription {

	// Best-effort: an empty ServerKey just means the client can't decrypt push payloads yet,
	// not a request failure.
	serverKey, _ := factory.WebPush().PublicKey(session)

	return object.WebPushSubscription{
		ID:        endpoint,
		Endpoint:  endpoint,
		Standard:  true,
		Alerts:    alerts,
		ServerKey: serverKey,
	}
}

// requestedAlerts copies a PostPushSubscription's requested alert flags into the response shape.
func requestedAlerts(t txn.PostPushSubscription) object.WebPushSubscriptionAlerts {

	// Read through the transaction, so this never restates toot's anonymous struct and its tags
	alerts := t.Data.Alerts

	return object.WebPushSubscriptionAlerts{
		Mention:       alerts.Mention,
		Quote:         alerts.Quote,
		Status:        alerts.Status,
		Reblog:        alerts.Reblog,
		Follow:        alerts.Follow,
		FollowRequest: alerts.FollowRequest,
		Favourite:     alerts.Favourite,
		Poll:          alerts.Poll,
		Update:        alerts.Update,
		QuotedUpdate:  alerts.QuotedUpdate,
		AdminSignUp:   alerts.AdminSignUp,
		AdminReport:   alerts.AdminReport,
	}
}

// updatedAlerts copies a PutPushSubscription's requested alert flags into the response shape.
func updatedAlerts(t txn.PutPushSubscription) object.WebPushSubscriptionAlerts {

	// Read through the transaction, so this never restates toot's anonymous struct and its tags
	alerts := t.Data.Alerts

	return object.WebPushSubscriptionAlerts{
		Mention:       alerts.Mention,
		Quote:         alerts.Quote,
		Status:        alerts.Status,
		Reblog:        alerts.Reblog,
		Follow:        alerts.Follow,
		FollowRequest: alerts.FollowRequest,
		Favourite:     alerts.Favourite,
		Poll:          alerts.Poll,
		Update:        alerts.Update,
		QuotedUpdate:  alerts.QuotedUpdate,
		AdminSignUp:   alerts.AdminSignUp,
		AdminReport:   alerts.AdminReport,
	}
}

// allAlertsEnabled is GetPushSubscription's answer for a client asking which alerts are on.
func allAlertsEnabled() object.WebPushSubscriptionAlerts {

	return object.WebPushSubscriptionAlerts{
		Mention: true, Quote: true, Status: true, Reblog: true, Follow: true,
		FollowRequest: true, Favourite: true, Poll: true, Update: true,
		QuotedUpdate: true, AdminSignUp: true, AdminReport: true,
	}
}
