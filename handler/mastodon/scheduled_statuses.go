package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/scheduled_statuses/
func GetScheduledStatuses(serverFactory *server.Factory) func(model.Authorization, txn.GetScheduledStatuses) ([]object.ScheduledStatus, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetScheduledStatuses) ([]object.ScheduledStatus, toot.PageInfo, error) {
		return []object.ScheduledStatus{}, toot.PageInfo{}, nil
	}
}

// GetScheduledStatus implements the Mastodon "get scheduled status" endpoint, and always returns an empty status
func GetScheduledStatus(serverFactory *server.Factory) func(model.Authorization, txn.GetScheduledStatus) (object.ScheduledStatus, error) {

	return func(model.Authorization, txn.GetScheduledStatus) (object.ScheduledStatus, error) {
		return object.ScheduledStatus{}, nil

	}
}

// PutScheduledStatus implements the Mastodon "update scheduled status" endpoint as a no-op
func PutScheduledStatus(serverFactory *server.Factory) func(model.Authorization, txn.PutScheduledStatus) (object.ScheduledStatus, error) {

	return func(model.Authorization, txn.PutScheduledStatus) (object.ScheduledStatus, error) {
		return object.ScheduledStatus{}, nil

	}
}

// DeleteScheduledStatus implements the Mastodon "delete scheduled status" endpoint as a no-op
func DeleteScheduledStatus(serverFactory *server.Factory) func(model.Authorization, txn.DeleteScheduledStatus) (struct{}, error) {

	return func(model.Authorization, txn.DeleteScheduledStatus) (struct{}, error) {
		return struct{}{}, nil
	}
}
