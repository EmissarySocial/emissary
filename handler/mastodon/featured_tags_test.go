package mastodon

import (
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestFeaturedTag_Details confirms a profile hashtag carries the ID, name, link and counts clients need
func TestFeaturedTag_Details(t *testing.T) {

	user := model.User{ProfileURL: "https://example.com/@alice", TagURL: "/search?q="}

	tag := featuredTag(&user, "Gardening", 3, 1789000000)

	require.Equal(t, "Gardening", tag.ID)
	require.Equal(t, "Gardening", tag.Name)
	require.Equal(t, "https://example.com/search?q=%23Gardening", tag.URL)
	require.Equal(t, "3", tag.StatusesCount)
	require.Equal(t, "2026-09-10", tag.LastStatusAt)
}

// TestFeaturedTag_NoTagLinks confirms the profile stands in for the link when the template has none
func TestFeaturedTag_NoTagLinks(t *testing.T) {

	user := model.User{ProfileURL: "https://example.com/@alice"}

	tag := featuredTag(&user, "Gardening", 0, 0)

	require.Equal(t, "https://example.com/@alice", tag.URL)
	require.Equal(t, "0", tag.StatusesCount)
	require.Empty(t, tag.LastStatusAt)
}

// TestFeaturedTag_JSON confirms the count is sent as a string and an unknown date is left out, as Mastodon does
func TestFeaturedTag_JSON(t *testing.T) {

	user := model.User{ProfileURL: "https://example.com/@alice"}

	result, err := json.Marshal(featuredTag(&user, "Gardening", 0, 0))

	require.NoError(t, err)
	require.JSONEq(t, `{"id":"Gardening","name":"Gardening","url":"https://example.com/@alice","statuses_count":"0"}`, string(result))
}
