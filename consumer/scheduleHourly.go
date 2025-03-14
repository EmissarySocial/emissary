package consumer

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

func ScheduleHourly(serverFactory ServerFactory) queue.Result {

	const location = "consumer.ScheduleHourly"
	log.Trace().Str("location", location).Msg("Running Hourly Tasks...")

	// Hourly tasks for each domain
	for factory := range serverFactory.RangeDomains() {

		// Add "Shuffle" tasks to the queue
		task := queue.NewTask(
			"Shuffle",
			mapof.Any{"host": factory.Hostname()},
			queue.WithPriority(300),
		)

		if err := serverFactory.Queue().Publish(task); err != nil {
			return queue.Error(err)
		}
	}

	return queue.Success()
}
