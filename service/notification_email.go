package service

import (
	"math"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// notificationEmailPageSize is the most notifications that one unread email lists
const notificationEmailPageSize = 60

// notificationEmailer is the part of the DomainEmail service that the unread email needs
type notificationEmailer interface {
	IsConfigured() bool
	SendUnreadNotifications(user *model.User, notifications []model.Notification, moreCount int64) error
}

// UsersWithUnread returns the ID of every User who holds an unread Notification created at or
// before the cutoff (Unix epoch MILLISECONDS)
func (service *Notification) UsersWithUnread(session data.Session, beforeMillis int64) ([]primitive.ObjectID, error) {
	return queries.NotificationUsersWithUnread(session, beforeMillis)
}

// EmailUnread sends the User one email listing their unread Notifications, when they are owed one
// at the given moment (Unix epoch MILLISECONDS). It returns TRUE when an email was sent.
func (service *Notification) EmailUnread(session data.Session, user *model.User, nowMillis int64) (bool, error) {

	const location = "service.Notification.EmailUnread"

	// RULE: A Domain that cannot send email claims nothing, so the email goes out once it can
	if !service.emailService.IsConfigured() {
		return false, nil
	}

	// RULE: The User's own span and watermark decide which Notifications can cause an email
	after, before, isEnabled := user.NotificationEmailWindow(nowMillis)

	if !isEnabled {
		return false, nil
	}

	// Built fresh for each query, because chaining onto a shared expression can alias its terms
	unread := func() exp.Expression {
		return exp.Equal("userId", user.UserID).AndEqual("readDate", int64(math.MaxInt64))
	}

	owed, err := service.Count(session, unread().AndGreaterThan("createDate", after).AndLessOrEqual("createDate", before))

	if err != nil {
		return false, derp.Wrap(err, location, "Counting the Notifications that are owed an email", user.UserID)
	}

	if owed == 0 {
		return false, nil
	}

	// The email lists every unread Notification, newest first, and counts the ones past the first page
	notifications, err := service.Query(session, unread(), option.SortDesc("createDate"), option.MaxRows(notificationEmailPageSize))

	if err != nil {
		return false, derp.Wrap(err, location, "Loading unread Notifications", user.UserID)
	}

	if len(notifications) == 0 {
		return false, nil
	}

	unreadCount, err := service.Count(session, unread())

	if err != nil {
		return false, derp.Wrap(err, location, "Counting unread Notifications", user.UserID)
	}

	// RULE: Claim the watermark before sending.  A task that matches nothing lost the race to
	// another server, which sends the email itself.
	previous := user.NotificationEmailDate
	newest := notifications[0].CreateDate

	claimed, err := queries.ClaimNotificationEmail(session, user.UserID, previous, newest)

	if err != nil {
		return false, derp.Wrap(err, location, "Claiming the notification email", user.UserID)
	}

	if !claimed {
		return false, nil
	}

	user.NotificationEmailDate = newest

	// RULE: A Notification whose actor the User's rules now hide is left out.  If none are left,
	// nothing is sent and the watermark stays moved.
	service.ruleService.LabelNotifications(session, user.UserID, notifications)
	visible := visibleNotifications(notifications)

	if len(visible) == 0 {
		return false, nil
	}

	moreCount := max(unreadCount-int64(len(notifications)), 0)

	if err := service.emailService.SendUnreadNotifications(user, visible, moreCount); err != nil {

		// RULE: A failed send puts the watermark back, so the next run tries again
		if _, restoreErr := queries.ClaimNotificationEmail(session, user.UserID, newest, previous); restoreErr != nil {
			derp.Report(derp.Wrap(restoreErr, location, "Restoring the notification email watermark", user.UserID))
		} else {
			user.NotificationEmailDate = previous
		}

		// RULE: The error names the User by ID, never by the record or the email address
		return false, derp.Wrap(err, location, "Sending unread notifications email", user.UserID)
	}

	// You've got mail
	return true, nil
}

// visibleNotifications returns the Notifications that the viewer's rules do not hide
func visibleNotifications(notifications []model.Notification) []model.Notification {

	result := make([]model.Notification, 0, len(notifications))

	for _, notification := range notifications {
		if !notification.Labels.IsHidden() {
			result = append(result, notification)
		}
	}

	return result
}
