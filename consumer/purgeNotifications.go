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

// PurgeNotifications removes notifications that are older than notificationRetentionDays.
func PurgeNotifications(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.PurgeNotifications"

	log.Trace().Msg("Task: PurgeNotifications")

	// journal.createDate is stored in Unix MILLISECONDS, so compute the cutoff in millis.
	cutoffMillis := time.Now().AddDate(0, 0, -notificationRetentionDays).UnixMilli()

	if err := factory.Notification().PurgeBefore(session, cutoffMillis); err != nil {
		return queue.Error(derp.Wrap(err, location, "Purging old notifications"))
	}

	return queue.Success()
}
