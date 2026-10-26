package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
)

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
