package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
)

// TestTrendingTagsLimit keeps the limit within Mastodon's bounds
func TestTrendingTagsLimit(t *testing.T) {
	require.Equal(t, trendingTagsDefault, trendingTagsLimit(""))
	require.Equal(t, trendingTagsDefault, trendingTagsLimit("nope"))
	require.Equal(t, trendingTagsDefault, trendingTagsLimit("-3"))
	require.Equal(t, 5, trendingTagsLimit("5"))
	require.Equal(t, trendingTagsMax, trendingTagsLimit("500"))
}

// TestSearchTagToTag names the hashtag by its normalized value and links it on this server
func TestSearchTagToTag(t *testing.T) {

	tag, ok := searchTagToTag(model.SearchTag{Name: "Indie Rock", Value: "indierock"}, "social.example")
	require.True(t, ok)
	require.Equal(t, "indierock", tag.Name)
	require.Equal(t, "https://social.example/tags/indierock", tag.URL)
	require.NotNil(t, tag.History, "history is an empty list, not null")

	_, ok = searchTagToTag(model.SearchTag{Name: "Nothing"}, "social.example")
	require.False(t, ok)
}

// TestStubsRefuseInsteadOfFaking confirms the endpoints for things Emissary lacks refuse instead of answering success
func TestStubsRefuseInsteadOfFaking(t *testing.T) {

	_, err := PostTag_Follow(nil)(model.Authorization{}, txn.PostTag_Follow{})
	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err))

	_, err = GetScheduledStatus(nil)(model.Authorization{}, txn.GetScheduledStatus{})
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))

	_, err = DeleteScheduledStatus(nil)(model.Authorization{}, txn.DeleteScheduledStatus{})
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))

	_, err = PostAnnouncement_Dismiss(nil)(model.Authorization{}, txn.PostAnnouncement_Dismiss{})
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))
}
