package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

// TestNotificationTabTypes pins which notification types each notifications-page section lists
func TestNotificationTabTypes(t *testing.T) {

	// Messages & Replies casts the widest net, quotes included
	require.Equal(t, []string{model.NotificationTypeMention, model.NotificationTypeReply, model.NotificationTypeDirect, model.NotificationTypeQuote}, notificationTabTypes(model.NotificationTypeMention))

	// Likes folds in dislikes
	require.Equal(t, []string{model.NotificationTypeLike, model.NotificationTypeDislike}, notificationTabTypes(model.NotificationTypeLike))

	// Every other section lists only its own type
	require.Equal(t, []string{model.NotificationTypeAnnounce}, notificationTabTypes(model.NotificationTypeAnnounce))
	require.Equal(t, []string{model.NotificationTypeFollow}, notificationTabTypes(model.NotificationTypeFollow))

	// No section means every type
	require.Nil(t, notificationTabTypes(""))
}

// TestNotificationTabTypes_EveryTypeHasASection pins that every Notification type is listed by some
// section, so that no notification is stored where the page can never show it
func TestNotificationTabTypes_EveryTypeHasASection(t *testing.T) {

	listed := map[string]bool{}

	for _, section := range []string{model.NotificationTypeMention, model.NotificationTypeAnnounce, model.NotificationTypeLike, model.NotificationTypeFollow} {
		for _, notificationType := range notificationTabTypes(section) {
			listed[notificationType] = true
		}
	}

	for _, notificationType := range []string{model.NotificationTypeDirect, model.NotificationTypeMention, model.NotificationTypeReply, model.NotificationTypeQuote, model.NotificationTypeLike, model.NotificationTypeDislike, model.NotificationTypeAnnounce, model.NotificationTypeFollow} {
		require.True(t, listed[notificationType], "%s is listed by no section of the notifications page", notificationType)
	}
}

// TestNotificationTypeFilter pins the criteria built from a section name
func TestNotificationTypeFilter(t *testing.T) {
	require.Equal(t, exp.All(), notificationTypeFilter(""))
	require.Equal(t, exp.In("type", notificationTabTypes(model.NotificationTypeMention)), notificationTypeFilter(model.NotificationTypeMention))
}
