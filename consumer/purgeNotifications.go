package consumer

import (
	"time"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// notificationRetentionDays is the number of days a notification is kept before it is purged.
// The window is UNIFORM: read and unread notifications age out alike (see Notification.PurgeBefore).
const notificationRetentionDays = 90

// notificationCapPerUser is the maximum number of live notifications retained per User.  Beyond this,
// the daily purge trims the oldest (READ rows first) as a flood backstop that retention alone cannot
// provide -- a flood lands in minutes, retention purges in months (see the NOTIFICATION-FLOOD-CONTROL
// spec).  This is a product-tunable ceiling, not a technical limit; nobody scrolls past their
// two-thousandth notification.
const notificationCapPerUser = 2000

// PurgeNotifications removes notifications older than notificationRetentionDays, then trims any User
// holding more than notificationCapPerUser live notifications back down to the cap.
func PurgeNotifications(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.PurgeNotifications"

	log.Trace().Msg("Task: PurgeNotifications")

	// journal.createDate is stored in Unix MILLISECONDS, so compute the cutoff in millis.
	cutoffMillis := time.Now().AddDate(0, 0, -notificationRetentionDays).UnixMilli()

	// Retention first: age out everything past the window (this shrinks the set the cap must consider).
	if err := factory.Notification().PurgeBefore(session, cutoffMillis); err != nil {
		return queue.Error(derp.Wrap(err, location, "Purging old notifications"))
	}

	// Cap second: trim any User still over the per-user ceiling (the flood backstop).
	if err := factory.Notification().PurgeOverCap(session, notificationCapPerUser); err != nil {
		return queue.Error(derp.Wrap(err, location, "Enforcing per-user notification cap"))
	}

	return queue.Success()
}
