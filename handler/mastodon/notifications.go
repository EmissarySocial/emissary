package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// https://docs.joinmastodon.org/methods/notifications/
func GetNotifications(serverFactory *server.Factory) func(model.Authorization, txn.GetNotifications) ([]object.Notification, toot.PageInfo, error) {

	const location = "handler.mastodon.GetNotifications"

	return func(auth model.Authorization, t txn.GetNotifications) ([]object.Notification, toot.PageInfo, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return []object.Notification{}, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return []object.Notification{}, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Query this user's notifications with Mastodon paging.  DISLIKE has no Mastodon equivalent,
		// so it is excluded from this API.
		criteria := queryExpression(t).AndNotEqual("type", model.NotificationTypeDislike)

		notifications, err := factory.Notification().QueryByUserID(session, auth.UserID, criteria)

		if err != nil {
			return []object.Notification{}, toot.PageInfo{}, derp.Wrap(err, location, "Querying notifications")
		}

		return getSliceOfToots(notifications), getPageInfo(notifications), nil
	}
}

// GetNotification implements the Mastodon "get notification" endpoint
func GetNotification(serverFactory *server.Factory) func(model.Authorization, txn.GetNotification) (object.Notification, error) {

	const location = "handler.mastodon.GetNotification"

	return func(auth model.Authorization, t txn.GetNotification) (object.Notification, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		notificationID, err := primitive.ObjectIDFromHex(t.ID)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Invalid notification ID", t.ID, derp.WithBadRequest())
		}

		// LoadByID is scoped to auth.UserID, so it cannot return another user's notification.
		notification := model.NewNotification()

		if err := factory.Notification().LoadByID(session, auth.UserID, notificationID, &notification); err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Loading notification", t.ID)
		}

		return notification.Toot(), nil
	}
}

// PostNotifications_Clear implements the Mastodon "clear all notifications" endpoint
func PostNotifications_Clear(serverFactory *server.Factory) func(model.Authorization, txn.PostNotifications_Clear) (object.Notification, error) {

	const location = "handler.mastodon.PostNotifications_Clear"

	return func(auth model.Authorization, t txn.PostNotifications_Clear) (object.Notification, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		if err := factory.Notification().DeleteByUserID(session, auth.UserID, "Cleared via Mastodon API"); err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Clearing notifications")
		}

		return object.Notification{}, nil
	}
}

// PostNotification_Dismiss implements the Mastodon "dismiss notification" endpoint
func PostNotification_Dismiss(serverFactory *server.Factory) func(model.Authorization, txn.PostNotification_Dismiss) (object.Notification, error) {

	const location = "handler.mastodon.PostNotification_Dismiss"

	return func(auth model.Authorization, t txn.PostNotification_Dismiss) (object.Notification, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		notificationID, err := primitive.ObjectIDFromHex(t.ID)

		if err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Invalid notification ID", t.ID, derp.WithBadRequest())
		}

		// Load (scoped to auth.UserID) then delete, so a user can only dismiss their own notification.
		notification := model.NewNotification()

		if err := factory.Notification().LoadByID(session, auth.UserID, notificationID, &notification); err != nil {

			if derp.IsNotFound(err) {
				return object.Notification{}, nil // Idempotent: already gone.
			}

			return object.Notification{}, derp.Wrap(err, location, "Loading notification", t.ID)
		}

		if err := factory.Notification().Delete(session, &notification, "Dismissed via Mastodon API"); err != nil {
			return object.Notification{}, derp.Wrap(err, location, "Dismissing notification", t.ID)
		}

		return object.Notification{}, nil
	}
}
