package mastodon

import (
	"slices"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
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
		criteria = withNotificationTypeFilters(criteria, t.Types, t.ExcludeTypes)

		notifications, err := factory.Notification().QueryByUserID(session, auth.UserID, criteria, option.MaxRows(pageLimit(t.Limit)))

		if err != nil {
			return []object.Notification{}, toot.PageInfo{}, derp.Wrap(err, location, "Querying notifications")
		}

		return notificationsToToots(factory, session, auth, notifications), getPageInfo(notifications), nil
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

		return notificationToToot(factory, session, auth, notification), nil
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

// https://docs.joinmastodon.org/methods/notifications/#unread-count
func GetNotifications_UnreadCount(serverFactory *server.Factory) func(model.Authorization, txn.GetNotifications_UnreadCount) (object.NotificationsUnreadCount, error) {

	const location = "handler.mastodon.GetNotifications_UnreadCount"

	return func(auth model.Authorization, t txn.GetNotifications_UnreadCount) (object.NotificationsUnreadCount, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.NotificationsUnreadCount{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.NotificationsUnreadCount{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// t.Types/t.ExcludeTypes narrow the count the same way they narrow GetNotifications.
		// DISLIKE has no Mastodon equivalent and is never included.
		types := unreadCountTypes(t.Types, t.ExcludeTypes)

		count, err := factory.Notification().CountUnread(session, auth.UserID, types...)

		if err != nil {
			return object.NotificationsUnreadCount{}, derp.Wrap(err, location, "Counting unread notifications")
		}

		return object.NotificationsUnreadCount{Count: int(count)}, nil
	}
}

// https://docs.joinmastodon.org/methods/notifications/#get-policy
//
// Always answers "accept" -- Emissary delivers anything that clears block/mute rules, with
// no further classification to filter by.
func GetNotificationPolicy(serverFactory *server.Factory) func(model.Authorization, txn.GetNotificationPolicy) (object.NotificationPolicy, error) {

	return func(auth model.Authorization, t txn.GetNotificationPolicy) (object.NotificationPolicy, error) {

		const accept = "accept"

		return object.NotificationPolicy{
			ForNotFollowing:    accept,
			ForNotFollowers:    accept,
			ForNewAccounts:     accept,
			ForPrivateMentions: accept,
			ForLimitedAccounts: accept,
			ForBots:            accept,
			Summary:            object.NotificationPolicySummary{},
		}, nil
	}
}

// notificationsToToots converts Notifications to their Mastodon form, attaching each one's
// Status concurrently -- a MENTION/DIRECT needs a live remote fetch, and a page can hold several.
func notificationsToToots(factory *service.Factory, session data.Session, auth model.Authorization, notifications []model.Notification) []object.Notification {

	const maxConcurrent = 16

	result := make([]object.Notification, len(notifications))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, maxConcurrent)

	for index := range notifications {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int) {
			defer waitGroup.Done()
			defer func() { <-slots }()
			result[index] = notificationToToot(factory, session, auth, notifications[index])
		}(index)
	}

	waitGroup.Wait()

	// RULE: drop a status-bearing notification whose post couldn't be fetched, matching Mastodon
	return slices.DeleteFunc(result, func(notification object.Notification) bool {
		return notificationNeedsStatus(notification.Type) && notification.Status == nil
	})
}

// notificationNeedsStatus returns TRUE for notification types whose client rendering
// depends on an attached status.
func notificationNeedsStatus(notificationType string) bool {
	switch notificationType {
	case "mention", "favourite", "reblog", "status", "update", "poll":
		return true
	}
	return false
}

// notificationToToot converts a single Notification, attaching the Status it's about when
// there is one.
func notificationToToot(factory *service.Factory, session data.Session, auth model.Authorization, notification model.Notification) object.Notification {

	result := notification.Toot()

	switch {

	// REPLY/LIKE/DISLIKE/ANNOUNCE always carry a StreamID for one of the recipient's own Streams
	case !notification.StreamID.IsZero():

		stream := model.NewStream()

		if err := factory.Stream().LoadByID(session, notification.StreamID, &stream); err == nil {
			status := tootStream(factory, session, &stream)
			result.Status = &status
		}

	// MENTION/DIRECT have no local Stream; fetch it the same way a profile's posts are
	case notification.ObjectURL != "":

		if status, err := statusForPostURL(factory, session, auth, notification.ObjectURL, "handler.mastodon_notificationToToot"); err == nil {
			result.Status = &status
		}
	}

	return result
}

// mastodonNotificationTypesToInternal maps Mastodon type names from types[]/exclude_types[]
// to Emissary's internal Notification types. An unrecognized Mastodon type maps to nothing.
func mastodonNotificationTypesToInternal(mastodonTypes []string) []string {

	result := make([]string, 0, len(mastodonTypes)*2)

	for _, mastodonType := range mastodonTypes {

		switch mastodonType {

		case "mention":
			result = append(result, model.NotificationTypeDirect, model.NotificationTypeMention, model.NotificationTypeReply)

		case "favourite":
			result = append(result, model.NotificationTypeLike)

		case "reblog":
			result = append(result, model.NotificationTypeAnnounce)

		case "follow":
			result = append(result, model.NotificationTypeFollow)
		}
	}

	return result
}

// withNotificationTypeFilters applies GetNotifications' optional types/exclude_types filters
// to a query expression already built by queryExpression.
func withNotificationTypeFilters(criteria exp.Expression, types []string, excludeTypes []string) exp.Expression {

	if len(types) > 0 {
		criteria = criteria.AndIn("type", mastodonNotificationTypesToInternal(types))
	}

	if len(excludeTypes) > 0 {
		criteria = criteria.AndNotIn("type", mastodonNotificationTypesToInternal(excludeTypes))
	}

	return criteria
}

// unreadCountTypes builds CountUnread's type filter: an explicit types[] narrows to those
// types, otherwise exclude_types[] is resolved against every producible type.
func unreadCountTypes(types []string, excludeTypes []string) []string {

	if len(types) > 0 {
		return mastodonNotificationTypesToInternal(types)
	}

	if len(excludeTypes) == 0 {
		return nil
	}

	all := []string{
		model.NotificationTypeDirect,
		model.NotificationTypeMention,
		model.NotificationTypeReply,
		model.NotificationTypeLike,
		model.NotificationTypeAnnounce,
		model.NotificationTypeFollow,
	}

	excluded := mastodonNotificationTypesToInternal(excludeTypes)
	result := make([]string, 0, len(all))

	for _, t := range all {
		if !slices.Contains(excluded, t) {
			result = append(result, t)
		}
	}

	return result
}
