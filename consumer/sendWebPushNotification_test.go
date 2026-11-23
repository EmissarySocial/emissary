package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestNotificationTitle pins the Web Push title for every Notification type
func TestNotificationTitle(t *testing.T) {

	table := map[string]string{
		model.NotificationTypeDirect:   "Bob sent you a message",
		model.NotificationTypeMention:  "Bob mentioned you",
		model.NotificationTypeReply:    "Bob replied to you",
		model.NotificationTypeQuote:    "Bob quoted your post",
		model.NotificationTypeLike:     "Bob liked your post",
		model.NotificationTypeDislike:  "Bob disliked your post",
		model.NotificationTypeAnnounce: "Bob boosted your post",
		model.NotificationTypeFollow:   "Bob followed you",
		"BOGUS":                        "Bob",
	}

	for notificationType, expected := range table {
		notification := model.NewNotification()
		notification.Type = notificationType
		notification.Actor.Name = "Bob"
		require.Equal(t, expected, notificationTitle(&notification), notificationType)
	}
}

// TestNotificationURL_Quote pins that a quote opens the notifications page, where its Reject button is
func TestNotificationURL_Quote(t *testing.T) {

	notification := model.NewNotification()
	notification.Type = model.NotificationTypeQuote
	notification.ObjectURL = "https://remote.example/users/bob/statuses/1"

	require.Equal(t, "https://local.example/@me/notifications", notificationURL("https://local.example", &notification))
}
