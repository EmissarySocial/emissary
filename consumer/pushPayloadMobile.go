package consumer

import (
	"github.com/EmissarySocial/emissary/model"
)

// mobilePushPayload is the JSON that mobile apps decrypt and decode: Mastodon's push payload
type mobilePushPayload struct {
	AccessToken      string `json:"access_token"`
	NotificationID   int64  `json:"notification_id"`
	NotificationType string `json:"notification_type"`
	PreferredLocale  string `json:"preferred_locale"`
	Icon             string `json:"icon,omitempty"`
	Title            string `json:"title"`
	Body             string `json:"body"`
}

// newMobilePushPayload builds the mobile app payload for a Notification
func newMobilePushPayload(notification *model.Notification) mobilePushPayload {

	// RULE: Clients decode notification_id as an integer and Emissary's IDs are hex, so this sends
	// CreateDate instead. access_token stays empty: Emissary does not keep the bearer token.
	return mobilePushPayload{
		NotificationID:   notification.CreateDate,
		NotificationType: notification.MastodonType(),
		PreferredLocale:  "en",
		Icon:             notification.Actor.IconURL,
		Title:            notificationTitle(notification),
		Body:             notification.ObjectSummary,
	}
}
