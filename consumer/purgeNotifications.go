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

// notificationRetentionDays is the number of days a READ notification is kept before it is purged.
// Unread notifications are kept indefinitely.
const notificationRetentionDays = 90

// PurgeNotifications removes READ notifications that are older than notificationRetentionDays.
func PurgeNotifications(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.PurgeNotifications"

	log.Trace().Msg("Task: PurgeNotifications")

	// journal.createDate is stored in Unix MILLISECONDS, so compute the cutoff in millis.
	cutoffMillis := time.Now().AddDate(0, 0, -notificationRetentionDays).UnixMilli()

	if err := factory.Notification().PurgeReadBefore(session, cutoffMillis); err != nil {
		return queue.Error(derp.Wrap(err, location, "Purging old notifications"))
	}

	return queue.Success()
}
