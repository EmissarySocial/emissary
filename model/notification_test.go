package model

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestNotification verifies that every Notification property round-trips through the schema
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

// TestNotification_Channels pins the POLICY mapping from notification facts
// (Type, Subtype) to user settings channels
func TestNotification_Channels(t *testing.T) {

	// Every (Type, Subtype) combination is covered here, including the empty-subtype
	// fail-open rule. This is the single source of channel policy.

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

// TestNotification_IsConversation pins the ROUTING rule that decides which app a
// Notification opens
func TestNotification_IsConversation(t *testing.T) {

	// Only DIRECT routes to the Conversations app. Records written before DIRECT existed
	// are typed MENTION and must keep opening the public viewer.

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

// TestNotification_Channels_Direct guards the single most dangerous omission in this
// file: a DIRECT notification that yields no channels
func TestNotification_Channels_Direct(t *testing.T) {

	// A type falling through Channels() to the default returns NO channels, which makes
	// service.Notification.notify mark it born-read: no unread dot, no SSE, no Web Push.
	// A direct message that silently disappears is worse than a misrouted one.

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

// TestUserSchema_AllowsEveryNotificationChannel confirms the User schema accepts every
// channel, and writes it into the slice
func TestUserSchema_AllowsEveryNotificationChannel(t *testing.T) {

	// The settings form writes notificationChannels through this schema, so a channel the
	// enum rejects lets a user tick the box and watch the save quietly drop it. See
	// service.LookupProvider for the other half of the pairing.
	s := schema.New(UserSchema())

	for _, channel := range AllNotificationChannels() {
		user := NewUser()
		user.NotificationChannels = nil

		require.NoError(t, s.Set(&user, "notificationChannels.0", channel), "channel %q must be valid in the User schema", channel)

		// Set must GROW the nil slice, not just accept the value -- indexing straight into it would
		// panic on a Set that reported success without writing anything.
		require.Len(t, user.NotificationChannels, 1, "channel %q must be written into the slice", channel)
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

// TestNotificationChannels_SourceSweep fails if a NotificationChannel constant is
// declared without being added to AllNotificationChannels()
func TestNotificationChannels_SourceSweep(t *testing.T) {

	// The omission is silent and one-directional: the constant compiles, Channels() can
	// return it, but the User schema's enum rejects it, so the settings form refuses a
	// value the rest of the code considers legitimate.
	declaration := regexp.MustCompile(`(?m)^const (NotificationChannel[A-Za-z0-9]+) = "([A-Z_]+)"`)

	all := AllNotificationChannels()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	found := 0

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		source, err := os.ReadFile(entry.Name())
		require.NoError(t, err, entry.Name())

		for _, match := range declaration.FindAllStringSubmatch(string(source), -1) {
			found++
			require.Contains(t, all, match[2],
				"constant %s is not listed in AllNotificationChannels(); the User schema enum will reject it", match[1])
		}
	}

	// If the sweep finds nothing, the regex has drifted from the code, not the reverse.
	require.Positive(t, found, "expected to find at least one NotificationChannel constant")
	require.Len(t, all, found, "AllNotificationChannels() must list every declared channel, and nothing else")
}

// TestNotificationChannels_DefaultsAreASubset confirms every default channel is a
// channel the schema will accept
func TestNotificationChannels_DefaultsAreASubset(t *testing.T) {

	all := AllNotificationChannels()

	for _, channel := range DefaultNotificationChannels() {
		require.Contains(t, all, channel, "default channel %q is not a valid channel", channel)
	}
}
