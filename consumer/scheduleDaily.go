package consumer

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// ScheduleDaily queues the next day's worth of scheduled tasks for every Domain
func ScheduleDaily(serverFactory ServerFactory) queue.Result {

	const location = "consumer.ScheduleDaily"
	log.Trace().Str("location", location).Msg("Running Daily Tasks...")

	// Schedule the next batch of daily tasks
	if err := scheduler_MakeDailyTasks(serverFactory); err != nil {
		return queue.Error(err)
	}

	// Schedule the next batch of hourly tasks
	if err := scheduler_MakeHourlyTasks(serverFactory); err != nil {
		return queue.Error(err)
	}

	q := serverFactory.Queue()

	// Add a "Purge ActivityStream Cache" task to the queue
	q.NewTask("PurgeActivityStreamCache", mapof.Any{})

	// Add a "Purge Errors" task to the queue
	q.NewTask("PurgeErrors", mapof.Any{})

	// Add a "Purge Dome Log" task to the queue
	q.NewTask("PurgeDomeLog", mapof.Any{})

	// Daily tasks for each domain
	for factory := range serverFactory.RangeDomains() {

		// Add "Shuffle" tasks to the queue
		q.NewTask("Shuffle", mapof.Any{"hostname": factory.Hostname()})

		// Add "Recycle" tasks to the queue
		q.NewTask("RecycleDomain", mapof.Any{"hostname": factory.Hostname()})

		// Add "PurgeImports" tasks to the queue
		q.NewTask("PurgeImports", mapof.Any{"hostname": factory.Hostname()})

		// Add "PurgeNotifications" tasks to the queue
		q.NewTask("PurgeNotifications", mapof.Any{"hostname": factory.Hostname()})
	}

	// Stupendous.
	return queue.Success()
}
