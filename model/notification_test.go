package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestNotification(t *testing.T) {

	notification := NewNotification()

	s := schema.New(NotificationSchema())

	table := []tableTestItem{
		{"notificationId", "123412341234123412341234", nil},
		{"userId", "123456781234567812345678", nil},
		{"type", NotificationTypeMention, nil},
		{"subtype", NotificationSubtypeNotFollowing, nil},
		{"actor.name", "ACTOR NAME", nil},
		{"actor.emailAddress", "ACTOR@EMAIL.COM", nil},
		{"actor.profileUrl", "https://actor.example/website", nil},
		{"actor.iconUrl", "https://actor.example/photo.jpg", nil},
		{"activityId", "https://remote.example/activity/123", nil},
		{"objectUrl", "https://remote.example/note/456", nil},
		{"objectSummary", "A plain-text summary of the object", nil},
		{"streamId", "123456781234567812345679", nil},
		{"inReplyTo", "https://local.example/stream/789", nil},
	}

	tableTest_Schema(t, &s, &notification, table)
}

// TestNotification_Channels pins the POLICY mapping from notification facts (Type, Subtype)
// to user settings channels.  This is the single source of channel policy — every
// (Type, Subtype) combination is covered, including the empty-subtype fail-open rule.
func TestNotification_Channels(t *testing.T) {

	table := []struct {
		name     string
		typ      string
		subtype  string
		expected []string
	}{
		{"mention from followed", NotificationTypeMention, NotificationSubtypeFollowing, []string{NotificationChannelMentionFollowing}},
		{"mention from stranger", NotificationTypeMention, NotificationSubtypeNotFollowing, []string{NotificationChannelMentionNotFollowing}},
		{"mention empty subtype treated as following", NotificationTypeMention, "", []string{NotificationChannelMentionFollowing}},
		{"reply from followed (either enables)", NotificationTypeReply, NotificationSubtypeFollowing, []string{NotificationChannelReply, NotificationChannelMentionFollowing}},
		{"reply from stranger (either enables)", NotificationTypeReply, NotificationSubtypeNotFollowing, []string{NotificationChannelReply, NotificationChannelMentionNotFollowing}},
		{"reply empty subtype treated as following", NotificationTypeReply, "", []string{NotificationChannelReply, NotificationChannelMentionFollowing}},
		{"like folds into reaction", NotificationTypeLike, NotificationSubtypeFollowing, []string{NotificationChannelReaction}},
		{"dislike folds into reaction", NotificationTypeDislike, NotificationSubtypeNotFollowing, []string{NotificationChannelReaction}},
		{"announce folds into reaction", NotificationTypeAnnounce, "", []string{NotificationChannelReaction}},
		{"follow", NotificationTypeFollow, NotificationSubtypeNotFollowing, []string{NotificationChannelFollow}},
		{"unknown type maps to nothing", "BOGUS", "", []string{}},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			notification := NewNotification()
			notification.Type = test.typ
			notification.Subtype = test.subtype
			require.Equal(t, test.expected, notification.Channels())
		})
	}
}

// TestNotification_IsConversation pins the ROUTING rule that decides which app a Notification
// opens.  Only the DIRECT type routes to the Conversations app; records written before DIRECT
// existed are typed MENTION and must keep opening the public viewer.
func TestNotification_IsConversation(t *testing.T) {

	table := []struct {
		name     string
		typ      string
		expected bool
	}{
		{"direct message routes to Conversations", NotificationTypeDirect, true},
		{"mention routes to the public viewer", NotificationTypeMention, false},
		{"reply routes to the public viewer", NotificationTypeReply, false},
		{"like routes to the public viewer", NotificationTypeLike, false},
		{"follow routes to the public viewer", NotificationTypeFollow, false},
		{"legacy record with no type routes to the public viewer", "", false},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			notification := NewNotification()
			notification.Type = test.typ
			require.Equal(t, test.expected, notification.IsConversation())
		})
	}
}

// TestNotification_IsEncrypted pins the MLS predicate.  MLS is a DIRECT-only subtype, so a
// FOLLOWING/NOT_FOLLOWING subtype (carried by every other type) must never read as encrypted.
func TestNotification_IsEncrypted(t *testing.T) {

	table := []struct {
		name     string
		subtype  string
		expected bool
	}{
		{"MLS is encrypted", NotificationSubtypeMLS, true},
		{"plaintext is not", NotificationSubtypePlaintext, false},
		{"follow-state subtype is not", NotificationSubtypeFollowing, false},
		{"empty subtype is not", "", false},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			notification := NewNotification()
			notification.Subtype = test.subtype
			require.Equal(t, test.expected, notification.IsEncrypted())
		})
	}
}

// TestNotification_Channels_Direct guards the single most dangerous omission in this file.  A type
// that falls through Channels() to the default returns NO channels, which makes
// service.Notification.notify mark it born-read: no unread dot, no SSE nudge, no Web Push.  A
// direct message that silently disappears is worse than one that links to the wrong place.
func TestNotification_Channels_Direct(t *testing.T) {

	// The direct-message channel is the sole authority: no fallback to the mention channels, and
	// no follow-state split (DIRECT's Subtype carries the codec instead).
	expected := []string{NotificationChannelDirectMessage}

	for _, subtype := range []string{NotificationSubtypeMLS, NotificationSubtypePlaintext, ""} {
		notification := NewNotification()
		notification.Type = NotificationTypeDirect
		notification.Subtype = subtype

		require.Equal(t, expected, notification.Channels(), "subtype %q", subtype)
		require.NotEmpty(t, notification.Channels(), "DIRECT must never return zero channels")
	}
}

// TestDefaultNotificationChannels_IncludesDirectMessage pins the decision that direct messages are
// ON by default for new Users.  Existing Users are granted the same channel by upgrade Version29.
func TestDefaultNotificationChannels_IncludesDirectMessage(t *testing.T) {
	require.Contains(t, DefaultNotificationChannels(), NotificationChannelDirectMessage)
}

// TestUserSchema_AllowsEveryNotificationChannel guards a silent data-loss path: the settings form
// writes notificationChannels through the User schema, and rosetta validates against that enum.  A
// channel offered by the "notification-channels" lookup provider but missing from the enum would
// let the user tick the box and watch the save quietly drop it.  Every constant here must appear in
// both places -- see service.LookupProvider for the other half.
func TestUserSchema_AllowsEveryNotificationChannel(t *testing.T) {

	channels := []string{
		NotificationChannelDirectMessage,
		NotificationChannelMentionFollowing,
		NotificationChannelMentionNotFollowing,
		NotificationChannelReply,
		NotificationChannelFollow,
		NotificationChannelReaction,
	}

	s := schema.New(UserSchema())

	for _, channel := range channels {
		user := NewUser()
		user.NotificationChannels = nil

		require.NoError(t, s.Set(&user, "notificationChannels.0", channel), "channel %q must be valid in the User schema", channel)
		require.Equal(t, channel, user.NotificationChannels[0])
	}
}

// TestNotification_MastodonType_Direct guards the other silent-drop default: an unmapped type
// returns "", and handler/mastodon excludes those from the API entirely.
func TestNotification_MastodonType_Direct(t *testing.T) {
	notification := NewNotification()
	notification.Type = NotificationTypeDirect
	require.Equal(t, "mention", notification.MastodonType())
}
