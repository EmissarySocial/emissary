package consumer

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

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

		// Add "Recylce" tasks to the queue
		q.NewTask("RecycleDomain", mapof.Any{"host": factory.Hostname()})

	}

	// Stupendous.
	return queue.Success()
}
