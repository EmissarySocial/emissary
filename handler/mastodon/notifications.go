package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/notifications/
func GetNotifications(serverFactory *server.Factory) func(model.Authorization, txn.GetNotifications) ([]object.Notification, error) {

	return func(model.Authorization, txn.GetNotifications) ([]object.Notification, error) {

	}
}

func GetNotification(serverFactory *server.Factory) func(model.Authorization, txn.GetNotification) (object.Notification, error) {

	return func(model.Authorization, txn.GetNotification) (object.Notification, error) {

	}
}

func PostNotifications_Clear(serverFactory *server.Factory) func(model.Authorization, txn.PostNotifications_Clear) (object.Notification, error) {

	return func(model.Authorization, txn.PostNotifications_Clear) (object.Notification, error) {

	}
}

func PostNotification_Dismiss(serverFactory *server.Factory) func(model.Authorization, txn.PostNotification_Dismiss) (object.Notification, error) {

	return func(model.Authorization, txn.PostNotification_Dismiss) (object.Notification, error) {

	}
}
