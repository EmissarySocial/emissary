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
