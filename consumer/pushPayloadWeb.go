package consumer

import (
	"github.com/EmissarySocial/emissary/model"
)

// webPushPayload is the JSON delivered to the browser service worker's "push" handler.
type webPushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	URL   string `json:"url"`
}

// newWebPushPayload builds the browser payload for a Notification
func newWebPushPayload(host string, notification *model.Notification) webPushPayload {
	return webPushPayload{
		Title: notificationTitle(notification),
		Body:  notification.ObjectSummary,
		Icon:  notification.Actor.IconURL,
		URL:   notificationURL(host, notification),
	}
}

// notificationTitle builds the push notification title (actor + verb).
func notificationTitle(notification *model.Notification) string {

	actor := notification.Actor.Name
	if actor == "" {
		actor = notification.Actor.UsernameOrID()
	}

	switch notification.Type {
	case model.NotificationTypeDirect:
		return actor + " sent you a message"
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

	// RULE: A private message opens the Conversations app -- the only surface that owns it --
	// never a public one. This mirrors the same decision in the notification list templates;
	// model.Notification.IsConversation is the single source for both. See MESSAGE-AUDIENCE.md D2.
	if notification.IsConversation() {
		return host + "/@me/conversations"
	}

	// Reactions link to the local content; everything else opens the notifications section.
	switch notification.Type {
	case model.NotificationTypeLike, model.NotificationTypeDislike, model.NotificationTypeAnnounce:
		if notification.ObjectURL != "" {
			return notification.ObjectURL
		}
	}

	return host + "/@me/notifications"
}
