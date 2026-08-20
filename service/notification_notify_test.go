package service

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// A Create wrapper is judged by the delivering actor's identity keys AND the inner object's content
// keys -- so a TAG rule reaches the hashtags on the wrapped Note (D12/R16).
func TestNotificationMatchKeys_CreateWithTags(t *testing.T) {

	activity := streams.NewDocument(map[string]any{
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyActor: "https://origin.example/@author",
		vocab.PropertyObject: map[string]any{
			vocab.PropertyID:           "https://origin.example/tagged",
			vocab.PropertyAttributedTo: "https://origin.example/@author",
			vocab.PropertyTag: map[string]any{
				vocab.PropertyType: vocab.LinkTypeHashtag,
				vocab.PropertyName: "#qatest",
			},
		},
	})

	keys := notificationMatchKeys(activity)

	require.Contains(t, keys, "ACTOR:https://origin.example/@author")
	require.Contains(t, keys, "DOMAIN:origin.example")
	require.Contains(t, keys, "TAG:qatest")
}

// A Follow whose object is a bare URL contributes actor keys only -- the string object is never
// fetched, and produces no keys of its own.
func TestNotificationMatchKeys_FollowBareObject(t *testing.T) {

	activity := streams.NewDocument(map[string]any{
		vocab.PropertyType:   vocab.ActivityTypeFollow,
		vocab.PropertyActor:  "https://origin.example/@follower",
		vocab.PropertyObject: "https://local.example/@me",
	})

	keys := notificationMatchKeys(activity)

	require.Contains(t, keys, "ACTOR:https://origin.example/@follower")
	require.NotContains(t, keys, "ACTOR:https://local.example/@me")

	for _, key := range keys {
		require.False(t, strings.HasPrefix(key, "TAG:"), key)
	}
}

// TestReactionType verifies that each reaction activity maps to its Notification type, and that unknown types fall back to "Like"
func TestReactionType(t *testing.T) {
	require.Equal(t, model.NotificationTypeLike, reactionType(vocab.ActivityTypeLike))
	require.Equal(t, model.NotificationTypeDislike, reactionType(vocab.ActivityTypeDislike))
	require.Equal(t, model.NotificationTypeAnnounce, reactionType(vocab.ActivityTypeAnnounce))
	require.Equal(t, model.NotificationTypeLike, reactionType("SomethingElse")) // default
}

// TestPlainText_StripsTags verifies that HTML markup is removed from a notification summary
func TestPlainText_StripsTags(t *testing.T) {
	result := plainText("<p>Hello <b>world</b></p>")
	require.Equal(t, "Hello world", result)
}

// TestPlainText_UnescapesEntities verifies that HTML entities are decoded in a notification summary
func TestPlainText_UnescapesEntities(t *testing.T) {
	result := plainText("Fish &amp; chips")
	require.Equal(t, "Fish & chips", result)
}

// TestPlainText_Truncates verifies that an over-long summary is cut to the maximum length, plus an ellipsis
func TestPlainText_Truncates(t *testing.T) {
	long := strings.Repeat("a", objectSummaryMaxLength+50)
	result := plainText(long)

	// Truncated to the max length plus a single ellipsis rune
	require.Equal(t, objectSummaryMaxLength+1, len([]rune(result)))
	require.True(t, strings.HasSuffix(result, "…"))
}

// TestPlainText_ShortStringUnchanged verifies that a short summary passes through untouched
func TestPlainText_ShortStringUnchanged(t *testing.T) {
	require.Equal(t, "short", plainText("short"))
	require.Equal(t, "", plainText(""))
}
