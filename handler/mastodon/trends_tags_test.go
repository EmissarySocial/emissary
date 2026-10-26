package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
)

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
