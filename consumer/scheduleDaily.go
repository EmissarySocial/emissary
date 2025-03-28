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

	// Daily tasks for each domain
	for factory := range serverFactory.RangeDomains() {

		// Add "Recylce" tasks to the queue
		task := queue.NewTask(
			"RecycleDomain",
			mapof.Any{"host": factory.Hostname()},
			queue.WithPriority(300),
		)

		if err := serverFactory.Queue().Publish(task); err != nil {
			return queue.Error(err)
		}
	}

	// Stupendous.
	return queue.Success()
}
