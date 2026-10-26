package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
)

// TestPostStatus_RefusesScheduledPosts confirms a scheduled post is refused with a 422 before anything
// else runs (the nil server factory would panic if it got that far)
func TestPostStatus_RefusesScheduledPosts(t *testing.T) {

	_, err := PostStatus(nil)(model.Authorization{}, txn.PostStatus{Status: "later", ScheduledAt: "2030-01-01T00:00:00Z"})

	require.Error(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err))
}
