package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/scheduled_statuses/
func GetScheduledStatuses(serverFactory *server.Factory) func(model.Authorization, txn.GetScheduledStatuses) ([]object.ScheduledStatus, error) {

	return func(model.Authorization, txn.GetScheduledStatuses) ([]object.ScheduledStatus, error) {

	}
}

func GetScheduledStatus(serverFactory *server.Factory) func(model.Authorization, txn.GetScheduledStatus) (object.ScheduledStatus, error) {

	return func(model.Authorization, txn.GetScheduledStatus) (object.ScheduledStatus, error) {

	}
}

func PutScheduledStatus(serverFactory *server.Factory) func(model.Authorization, txn.PutScheduledStatus) (object.ScheduledStatus, error) {

	return func(model.Authorization, txn.PutScheduledStatus) (object.ScheduledStatus, error) {

	}
}

func DeleteScheduledStatus(serverFactory *server.Factory) func(model.Authorization, txn.DeleteScheduledStatus) (struct{}, error) {

	return func(model.Authorization, txn.DeleteScheduledStatus) (struct{}, error) {

	}
}
