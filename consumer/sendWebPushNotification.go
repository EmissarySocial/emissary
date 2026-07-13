package consumer

import (
	"encoding/json"
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

// webPushPayload is the JSON delivered to the service worker's "push" handler.
type webPushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	URL   string `json:"url"`
}

// SendWebPushNotification delivers a single Notification to all of the recipient User's Web Push
// subscriptions.  Subscriptions that the push service reports as gone (404/410) are pruned.
func SendWebPushNotification(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendWebPushNotification"

	log.Trace().Msg("Task: SendWebPushNotification")

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

		return queue.Error(derp.Wrap(err, location, "Unable to load notification", notificationID))
	}

	// Defense in depth: never push a read notification.  This covers suppressed records
	// (born read — see service.Notification.notify) and items the user already read
	// between enqueue and delivery.
	if notification.IsRead() {
		return queue.Success()
	}

	// Build the push payload once (server-side, so the service worker stays dumb).
	payload, err := json.Marshal(webPushPayload{
		Title: notificationTitle(&notification),
		Body:  notification.ObjectSummary,
		Icon:  notification.Actor.IconURL,
		URL:   notificationURL(factory.Host(), &notification),
	})

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Unable to marshal push payload"))
	}

	// Load the recipient's push subscriptions
	subscriptions, err := factory.PushSubscription().RangeByUserID(session, userID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load push subscriptions", userID))
	}

	for subscription := range subscriptions {
		sendWebPushToSubscription(factory, session, &subscription, payload)
	}

	return queue.Success()
}

// sendWebPushToSubscription delivers one payload to one subscription, pruning subscriptions
// that the push service reports as gone (404/410).  Errors are reported, never returned — a
// failure on one endpoint must not abort delivery to the others.
func sendWebPushToSubscription(factory *service.Factory, session data.Session, subscription *model.PushSubscription, payload []byte) {

	const location = "consumer.sendWebPushToSubscription"

	statusCode, err := factory.WebPush().Send(session, subscription.Endpoint, subscription.P256DH, subscription.Auth, payload)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to send Web Push", subscription.Endpoint))
		return
	}

	// 404/410 means the subscription is gone — prune it.
	if statusCode == http.StatusNotFound || statusCode == http.StatusGone {
		if err := factory.PushSubscription().DeleteByEndpoint(session, subscription.Endpoint, "expired"); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to delete expired push subscription", subscription.Endpoint))
		}
	}
}

// notificationTitle builds the push notification title (actor + verb).
func notificationTitle(notification *model.Notification) string {

	actor := notification.Actor.Name
	if actor == "" {
		actor = notification.Actor.UsernameOrID()
	}

	switch notification.Type {
	case model.NotificationTypeMention:
		return actor + " mentioned you"
	case model.NotificationTypeReply:
		return actor + " replied to you"
	case model.NotificationTypeLike:
		return actor + " liked your post"
	case model.NotificationTypeDislike:
		return actor + " disliked your post"
	case model.NotificationTypeAnnounce:
		return actor + " boosted your post"
	case model.NotificationTypeFollow:
		return actor + " followed you"
	default:
		return actor
	}
}

// notificationURL returns the URL that clicking the OS notification should open.
func notificationURL(host string, notification *model.Notification) string {

	// Reactions link to the local content; everything else opens the notifications section.
	switch notification.Type {
	case model.NotificationTypeLike, model.NotificationTypeDislike, model.NotificationTypeAnnounce:
		if notification.ObjectURL != "" {
			return notification.ObjectURL
		}
	}

	return host + "/@me/newsfeed/notifications"
}
