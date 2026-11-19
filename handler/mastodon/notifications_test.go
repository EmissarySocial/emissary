package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestMastodonNotificationTypesToInternal pins how Mastodon's notification type names map onto Emissary's
func TestMastodonNotificationTypesToInternal(t *testing.T) {

	require.Equal(t, []string{model.NotificationTypeDirect, model.NotificationTypeMention, model.NotificationTypeReply}, mastodonNotificationTypesToInternal([]string{"mention"}))
	require.Equal(t, []string{model.NotificationTypeLike}, mastodonNotificationTypesToInternal([]string{"favourite"}))
	require.Equal(t, []string{model.NotificationTypeAnnounce}, mastodonNotificationTypesToInternal([]string{"reblog"}))
	require.Equal(t, []string{model.NotificationTypeQuote}, mastodonNotificationTypesToInternal([]string{"quote"}))
	require.Equal(t, []string{model.NotificationTypeFollow}, mastodonNotificationTypesToInternal([]string{"follow"}))

	// Unknown names map to nothing, and several names combine in order
	require.Empty(t, mastodonNotificationTypesToInternal([]string{"poll", "quoted_update", ""}))
	require.Equal(t, []string{model.NotificationTypeQuote, model.NotificationTypeFollow}, mastodonNotificationTypesToInternal([]string{"quote", "poll", "follow"}))
}

// TestMastodonType_RoundTrips pins that every Notification type the API reports can also be filtered by that name
func TestMastodonType_RoundTrips(t *testing.T) {

	for _, notificationType := range []string{model.NotificationTypeDirect, model.NotificationTypeMention, model.NotificationTypeReply, model.NotificationTypeQuote, model.NotificationTypeLike, model.NotificationTypeAnnounce, model.NotificationTypeFollow} {
		notification := model.NewNotification()
		notification.Type = notificationType
		require.Contains(t, mastodonNotificationTypesToInternal([]string{notification.MastodonType()}), notificationType, notificationType)
	}
}

// TestUnreadCountTypes pins the type filter for the unread count
func TestUnreadCountTypes(t *testing.T) {

	// No filter at all counts every type
	require.Nil(t, unreadCountTypes(nil, nil))

	// types[] narrows to exactly those types
	require.Equal(t, []string{model.NotificationTypeQuote}, unreadCountTypes([]string{"quote"}, []string{"quote"}))

	// exclude_types[] removes from every producible type, quotes included
	require.Equal(t,
		[]string{model.NotificationTypeDirect, model.NotificationTypeMention, model.NotificationTypeReply, model.NotificationTypeLike, model.NotificationTypeAnnounce, model.NotificationTypeFollow},
		unreadCountTypes(nil, []string{"quote"}),
	)

	require.Contains(t, unreadCountTypes(nil, []string{"follow"}), model.NotificationTypeQuote)
}

// TestNotificationCarriesStatus confirms follows never carry a status while post notifications do.
func TestNotificationCarriesStatus(t *testing.T) {

	follow := model.NewNotification()
	follow.Type = model.NotificationTypeFollow
	require.False(t, notificationCarriesStatus(follow), "a follow is about a person, not a post")

	for _, kind := range []string{model.NotificationTypeMention, model.NotificationTypeReply, model.NotificationTypeDirect, model.NotificationTypeLike, model.NotificationTypeAnnounce} {
		notification := model.NewNotification()
		notification.Type = kind
		require.True(t, notificationCarriesStatus(notification), kind)
	}
}
