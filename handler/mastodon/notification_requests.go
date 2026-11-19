package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// GetNotificationRequests answers with no filtered notifications
// https://docs.joinmastodon.org/methods/notifications/#get-requests
func GetNotificationRequests(serverFactory *server.Factory) func(model.Authorization, txn.GetNotificationRequests) ([]object.NotificationRequest, toot.PageInfo, error) {

	// Emissary never holds a notification back, so there are no requests

	return func(model.Authorization, txn.GetNotificationRequests) ([]object.NotificationRequest, toot.PageInfo, error) {

		// Without this route, /notifications/requests reached GetNotification with the ID "requests"
		return []object.NotificationRequest{}, toot.PageInfo{}, nil
	}
}

// PostNotificationRequest_Accept answers that the request does not exist
// https://docs.joinmastodon.org/methods/notifications/#accept-request
func PostNotificationRequest_Accept(serverFactory *server.Factory) func(model.Authorization, txn.PostNotificationRequest_Accept) (struct{}, error) {

	// Emissary never holds a notification back, so there are no requests

	return func(_ model.Authorization, t txn.PostNotificationRequest_Accept) (struct{}, error) {
		return struct{}{}, derp.NotFound("handler.mastodon.PostNotificationRequest_Accept", "Notification request not found", t.ID)
	}
}

// PostNotificationRequest_Dismiss answers that the request does not exist
// https://docs.joinmastodon.org/methods/notifications/#dismiss-request
func PostNotificationRequest_Dismiss(serverFactory *server.Factory) func(model.Authorization, txn.PostNotificationRequest_Dismiss) (struct{}, error) {

	// Emissary never holds a notification back, so there are no requests

	return func(_ model.Authorization, t txn.PostNotificationRequest_Dismiss) (struct{}, error) {
		return struct{}{}, derp.NotFound("handler.mastodon.PostNotificationRequest_Dismiss", "Notification request not found", t.ID)
	}
}

// PostNotificationRequests_Accept succeeds with nothing to do, ignoring unknown IDs as Mastodon does
// https://docs.joinmastodon.org/methods/notifications/#accept-multiple-requests
func PostNotificationRequests_Accept(serverFactory *server.Factory) func(model.Authorization, txn.PostNotificationRequests_Accept) (struct{}, error) {

	return func(model.Authorization, txn.PostNotificationRequests_Accept) (struct{}, error) {
		return struct{}{}, nil
	}
}

// PostNotificationRequests_Dismiss succeeds with nothing to do, ignoring unknown IDs as Mastodon does
// https://docs.joinmastodon.org/methods/notifications/#dismiss-multiple-requests
func PostNotificationRequests_Dismiss(serverFactory *server.Factory) func(model.Authorization, txn.PostNotificationRequests_Dismiss) (struct{}, error) {

	return func(model.Authorization, txn.PostNotificationRequests_Dismiss) (struct{}, error) {
		return struct{}{}, nil
	}
}
