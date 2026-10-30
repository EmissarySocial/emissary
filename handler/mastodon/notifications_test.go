package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

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
