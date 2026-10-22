package consumer

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SendWebPushNotification delivers a Notification to every push subscription of its recipient,
// browsers and mobile apps alike, and prunes those the push service reports as gone (404/410)
func SendWebPushNotification(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendWebPushNotification"

	log.Trace().Msg("Task: SendWebPushNotification")

	// DIAGNOSTIC: this line proves the queue actually dispatched this task.  Its ABSENCE means the
	// task never ran, and the fault is in the queue rather than anywhere in Web Push.  Note that a
	// working SSE nudge cannot tell you this: SSE publishes with queue.WithInline(), which bypasses
	// the buffer and the storage provider entirely, so it runs even when the queue does not.
	log.Debug().
		Str("userId", args.GetString("userId")).
		Str("notificationId", args.GetString("notificationId")).
		Msg("Push: task STARTED")

	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid userId", args))
	}

	notificationID, err := primitive.ObjectIDFromHex(args.GetString("notificationId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid notificationId", args))
	}

	// Load the notification (scoped to the recipient User)
	notification := model.NewNotification()
	if err := factory.Notification().LoadByID(session, userID, notificationID, &notification); err != nil {

		// A deleted notification (e.g. an Undo arrived first) is not an error — nothing to push.
		if derp.IsNotFound(err) {
			return queue.Success()
		}

		return queue.Error(derp.Wrap(err, location, "Loading Notification", notificationID))
	}

	// Defense in depth: never push a read notification.  This covers suppressed records
	// (born read — see service.Notification.notify) and items the user already read
	// between enqueue and delivery.
	if notification.IsRead() {

		// DIAGNOSTIC: a notification that was SUPPRESSED by the recipient's channel settings is
		// "born read" (service.Notification.notify), so it lands here and delivers nothing.
		log.Debug().
			Str("notificationId", notificationID.Hex()).
			Msg("Push: notification is already READ -- nothing to deliver")

		return queue.Success()
	}

	// Build each payload once, server-side, so the browser and the mobile apps stay dumb.
	payloads, err := newPushPayloads(factory.Host(), &notification)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Building push payloads"))
	}

	// Load the recipient's push subscriptions
	subscriptions, err := factory.PushSubscription().RangeByUserID(session, userID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Loading PushSubscriptions", userID))
	}

	subscriptionCount := 0

	for subscription := range subscriptions {
		subscriptionCount = subscriptionCount + 1
		sendPushToSubscription(factory, session, &subscription, payloads)
	}

	// DIAGNOSTIC: zero subscriptions is not an error -- but it is the quietest way for a push to do
	// nothing whatsoever, because the loop above simply never runs.  Say so out loud.
	if subscriptionCount == 0 {
		log.Debug().
			Str("userId", userID.Hex()).
			Msg("Push: User has NO push subscriptions -- nothing to deliver")
	}

	return queue.Success()
}

// sendPushToSubscription delivers one notification to one subscription through the service that matches
// it (WebPush for a browser, MobilePush for a mobile app), pruning one the push service reports as gone
func sendPushToSubscription(factory *service.Factory, session data.Session, subscription *model.PushSubscription, payloads pushPayloads) {

	// Errors are reported, never returned: a failure on one endpoint must not stop delivery to the others

	const location = "consumer.sendPushToSubscription"

	send, payload := factory.WebPush().Send, payloads.web

	if subscription.IsMobile() {
		send, payload = factory.MobilePush().Send, payloads.mobile
	}

	statusCode, err := send(session, subscription.Endpoint, subscription.P256DH, subscription.Auth, payload)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Sending push", subscription.Endpoint))
		return
	}

	// 404/410 means the subscription is gone — prune it.
	if statusCode == http.StatusNotFound || statusCode == http.StatusGone {

		log.Debug().
			Int("statusCode", statusCode).
			Str("endpoint", subscription.Endpoint).
			Msg("Push: subscription is GONE -- pruning it")

		if err := factory.PushSubscription().DeleteByEndpoint(session, subscription.Endpoint, "expired"); err != nil {
			derp.Report(derp.Wrap(err, location, "Deleting expired PushSubscription", subscription.Endpoint))
		}

		return
	}

	// RULE: Any other non-2xx is the push service REFUSING to deliver -- a rejected VAPID JWT, an
	// oversized payload, rate limiting. Both services log its stated reason; report it here too,
	// so a refused push reaches the error log instead of looking like one never attempted.
	if (statusCode < 200) || (statusCode > 299) {
		derp.Report(derp.Internal(location, "Web Push service refused delivery", subscription.Endpoint, statusCode))
		return
	}

	log.Debug().
		Int("statusCode", statusCode).
		Str("endpoint", subscription.Endpoint).
		Msg("Push: DELIVERED to push service")
}
