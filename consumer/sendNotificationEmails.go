package consumer

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// notificationEmailMinimumAge is the shortest span a User can choose before an unread
// notification is emailed, so no younger notification can make anyone owed an email
const notificationEmailMinimumAge = time.Hour

// SendNotificationEmails emails every User on a Domain who is owed a list of unread notifications
func SendNotificationEmails(factory *service.Factory, _ mapof.Any) queue.Result {

	const location = "consumer.SendNotificationEmails"

	log.Trace().Str("hostname", factory.Hostname()).Msg("Task: SendNotificationEmails")

	// RULE: A Domain with no SMTP connection sends nothing and claims nothing
	if !factory.Email().IsConfigured() {
		return queue.Success()
	}

	// RULE: No transaction.  Each claim stands alone, and one email per User would soon outlast
	// MongoDB's 60-second limit on a transaction.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	session, err := factory.Server().Session(ctx)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Opening database session", factory.Hostname()))
	}

	defer session.Close()

	now := time.Now()
	notificationService := factory.Notification()

	userIDs, err := notificationService.UsersWithUnread(session, now.Add(-notificationEmailMinimumAge).UnixMilli())

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Finding Users with unread notifications", factory.Hostname()))
	}

	userService := factory.User()

	for _, userID := range userIDs {

		user := model.NewUser()

		if err := userService.LoadByID(session, userID, &user); err != nil {

			// A User deleted since their notifications arrived is owed nothing
			if derp.IsNotFound(err) {
				continue
			}

			derp.Report(derp.Wrap(err, location, "Loading User", userID))
			continue
		}

		// RULE: One User's failure is reported and the rest still get their email.  Returning the
		// error would retry the whole Domain, and the Users already mailed are not owed a second.
		if _, err := notificationService.EmailUnread(session, &user, now.UnixMilli()); err != nil {
			derp.Report(derp.Wrap(err, location, "Emailing unread notifications", userID))
		}
	}

	// Neither snow nor rain
	return queue.Success()
}
