package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
)

// TestGetNotificationRequests_Empty confirms the filtered-notification list is empty rather than an error.
func TestGetNotificationRequests_Empty(t *testing.T) {

	requests, pageInfo, err := GetNotificationRequests(nil)(model.Authorization{}, txn.GetNotificationRequests{})
	require.NoError(t, err)
	require.NotNil(t, requests)
	require.Empty(t, requests)
	require.Empty(t, pageInfo.MaxID)
}

// TestNotificationRequest_SingleIsNotFound confirms accepting or dismissing one request answers 404,
// since Emissary never holds a notification back
func TestNotificationRequest_SingleIsNotFound(t *testing.T) {

	_, err := PostNotificationRequest_Accept(nil)(model.Authorization{}, txn.PostNotificationRequest_Accept{ID: "abc"})
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))

	_, err = PostNotificationRequest_Dismiss(nil)(model.Authorization{}, txn.PostNotificationRequest_Dismiss{ID: "abc"})
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))
}

// TestNotificationRequests_BulkSucceeds confirms the bulk versions succeed, ignoring unknown IDs as Mastodon does
func TestNotificationRequests_BulkSucceeds(t *testing.T) {

	_, err := PostNotificationRequests_Accept(nil)(model.Authorization{}, txn.PostNotificationRequests_Accept{ID: []string{"abc"}})
	require.NoError(t, err)

	_, err = PostNotificationRequests_Dismiss(nil)(model.Authorization{}, txn.PostNotificationRequests_Dismiss{ID: []string{"abc"}})
	require.NoError(t, err)
}

// TestGetAccount_Collections_Empty confirms an account's collections are an empty list, not an error
func TestGetAccount_Collections_Empty(t *testing.T) {

	collections, _, err := GetAccount_Collections(nil)(model.Authorization{}, txn.GetAccount_Collections{AccountID: "abc"})
	require.NoError(t, err)
	require.NotNil(t, collections)
	require.Empty(t, collections)
}
