package service

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

func TestReactionType(t *testing.T) {
	require.Equal(t, model.NotificationTypeLike, reactionType(vocab.ActivityTypeLike))
	require.Equal(t, model.NotificationTypeDislike, reactionType(vocab.ActivityTypeDislike))
	require.Equal(t, model.NotificationTypeAnnounce, reactionType(vocab.ActivityTypeAnnounce))
	require.Equal(t, model.NotificationTypeLike, reactionType("SomethingElse")) // default
}

func TestPlainText_StripsTags(t *testing.T) {
	result := plainText("<p>Hello <b>world</b></p>")
	require.Equal(t, "Hello world", result)
}

func TestPlainText_UnescapesEntities(t *testing.T) {
	result := plainText("Fish &amp; chips")
	require.Equal(t, "Fish & chips", result)
}

func TestPlainText_Truncates(t *testing.T) {
	long := strings.Repeat("a", objectSummaryMaxLength+50)
	result := plainText(long)

	// Truncated to the max length plus a single ellipsis rune
	require.Equal(t, objectSummaryMaxLength+1, len([]rune(result)))
	require.True(t, strings.HasSuffix(result, "…"))
}

func TestPlainText_ShortStringUnchanged(t *testing.T) {
	require.Equal(t, "short", plainText("short"))
	require.Equal(t, "", plainText(""))
}
