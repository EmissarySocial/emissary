package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
)

// TestUnsupportedErrors confirms creating something Emissary lacks is a 422, and asking for one is a 404
func TestUnsupportedErrors(t *testing.T) {

	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(errNotSupported("test", "Filters")))
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(errNoSuch("test", "Poll")))

	// The handlers reach these before any database work, so a nil server factory is enough
	_, err := PostFilter(nil)(model.Authorization{}, txn.PostFilter{})
	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err))

	_, err = GetPoll(nil)(model.Authorization{}, txn.GetPoll{})
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))

	_, err = PostFollowRequest_Authorize(nil)(model.Authorization{}, txn.PostFollowRequest_Authorize{})
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))

	_, err = PostAccount_Pin(nil)(model.Authorization{}, txn.PostAccount_Pin{})
	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err))
}

// TestPostStatus_RefusesPolls confirms a post or edit carrying a poll is refused before anything is saved
func TestPostStatus_RefusesPolls(t *testing.T) {

	post := txn.PostStatus{Status: "which?"}
	post.Poll.Options = []string{"a", "b"}
	_, err := PostStatus(nil)(model.Authorization{}, post)
	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err))

	edit := txn.PutStatus{ID: "x", Status: "which?"}
	edit.Poll.Options = []string{"a", "b"}
	_, err = PutStatus(nil)(model.Authorization{}, edit)
	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err))
}

// TestMovedAccount confirms a moved actor points at a placeholder for its new account, and others at nothing
func TestMovedAccount(t *testing.T) {

	require.Nil(t, movedAccount("https://my.example", streams.NewDocument(map[string]any{"id": "https://a.example/users/x"})))
	require.Nil(t, movedAccount("https://my.example", streams.NewDocument(map[string]any{"movedTo": "not a url"})))

	moved := movedAccount("https://my.example", streams.NewDocument(map[string]any{"movedTo": "https://new.example/users/ben"}))
	require.NotNil(t, moved)
	require.Equal(t, "https://new.example/users/ben", moved.URL)
	require.NotEmpty(t, moved.Avatar, "the placeholder carries a default avatar, so apps can decode it")
}
