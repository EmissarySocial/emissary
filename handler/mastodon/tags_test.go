package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
)

// TestTagFollows_RequireSignIn refuses tag follows from callers who aren't signed in, before touching the database
func TestTagFollows_RequireSignIn(t *testing.T) {

	_, err := PostTag_Follow(nil)(model.Authorization{}, txn.PostTag_Follow{ID: "cats"})
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err))

	_, err = PostTag_Unfollow(nil)(model.Authorization{}, txn.PostTag_Unfollow{ID: "cats"})
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err))

	_, _, err = GetFollowedTags(nil)(model.Authorization{}, txn.GetFollowedTags{})
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err))
}

// TestFollowedTag describes a tag on this server, with its link and follow state
func TestFollowedTag(t *testing.T) {

	tag := followedTag("example.com", "cat lovers", true)

	require.Equal(t, "cat lovers", tag.Name)
	require.Equal(t, "https://example.com/tags/cat%20lovers", tag.URL)
	require.True(t, tag.Following)
	require.NotNil(t, tag.History)
}

// TestFollowedTagsLimit keeps the page size within Mastodon's bounds
func TestFollowedTagsLimit(t *testing.T) {
	require.Equal(t, int64(followedTagsDefault), followedTagsLimit(0))
	require.Equal(t, int64(followedTagsDefault), followedTagsLimit(-5))
	require.Equal(t, int64(20), followedTagsLimit(20))
	require.Equal(t, int64(followedTagsMax), followedTagsLimit(1000))
}
