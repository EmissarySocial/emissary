package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

		if err := factory.PushSubscription().Upsert(session, auth.UserID, endpoint, t.Subscription.Keys.P256dh, t.Subscription.Keys.Auth, ""); err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Saving subscription")
		}

		// RULE: Emissary has nowhere to persist per-type alert preferences, so every notification
		// that clears the recipient's rules generates a push regardless. Echo the requested
		// alerts back rather than silently ignore them.
		return pushSubscriptionToot(factory, session, endpoint, requestedAlerts(t.Data.Alerts)), nil
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

		sub, err := loadNewestPushSubscription(factory, session, auth.UserID)

		if err != nil {
			return object.WebPushSubscription{}, derp.Wrap(err, location, "Loading subscription")
		}

		// RULE: alerts aren't stored (see PostPushSubscription); reporting them all enabled
		// matches what actually happens, since nothing filters push delivery today.
		return pushSubscriptionToot(factory, session, sub.Endpoint, allAlertsEnabled()), nil
	}
}

// https://docs.joinmastodon.org/methods/push/#update
//
// Emissary has no field to persist per-type alert preferences, so this refuses rather than
// claim a change that can't take effect.
func PutPushSubscription(serverFactory *server.Factory) func(model.Authorization, txn.PutPushSubscription) (object.WebPushSubscription, error) {

	return func(auth model.Authorization, t txn.PutPushSubscription) (object.WebPushSubscription, error) {
		return object.WebPushSubscription{}, derp.NotImplemented("handler.mastodon.PutPushSubscription")
	}
}

// https://docs.joinmastodon.org/methods/push/#delete
//
// RULE: Mastodon scopes a subscription to the token that created it, but model.PushSubscription
// tracks no grant ID, so this removes every subscription the User has.
func DeletePushSubscription(serverFactory *server.Factory) func(model.Authorization, txn.DeletePushSubscription) (struct{}, error) {

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

		if err := factory.PushSubscription().DeleteByUserID(session, auth.UserID, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Deleting subscription")
		}

		return struct{}{}, nil
	}
}

// loadNewestPushSubscription returns the most recently created PushSubscription for a User.
func loadNewestPushSubscription(factory *service.Factory, session data.Session, userID primitive.ObjectID) (model.PushSubscription, error) {

	const location = "handler.mastodon.loadNewestPushSubscription"

	subs, err := factory.PushSubscription().Query(session, exp.Equal("userId", userID), option.SortDesc("createDate"), option.MaxRows(1))

	if err != nil {
		return model.PushSubscription{}, derp.Wrap(err, location, "Querying subscriptions")
	}

	if len(subs) == 0 {
		return model.PushSubscription{}, derp.NotFound(location, "No push subscription for this User")
	}

	return subs[0], nil
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
func requestedAlerts(alerts struct {
	Mention       bool `form:"mention"`
	Quote         bool `form:"quote"`
	Status        bool `form:"status"`
	Reblog        bool `form:"reblog"`
	Follow        bool `form:"follow"`
	FollowRequest bool `form:"follow_request"`
	Favourite     bool `form:"favourite"`
	Poll          bool `form:"poll"`
	Update        bool `form:"update"`
	QuotedUpdate  bool `form:"quoted_update"`
	AdminSignUp   bool `form:"admin.sign_up"`
	AdminReport   bool `form:"admin.report"`
}) object.WebPushSubscriptionAlerts {

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
