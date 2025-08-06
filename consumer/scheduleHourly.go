package consumer

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

func ScheduleHourly(serverFactory ServerFactory) queue.Result {

	const location = "consumer.ScheduleHourly"
	log.Trace().Str("location", location).Msg("Running Hourly Tasks...")

	q := serverFactory.Queue()

	// Hourly tasks for each domain
	for factory := range serverFactory.RangeDomains() {

		// Schedule "SearchNotifier" tasks every hour
		q.Enqueue <- queue.NewTask(
			"SearchNotifier",
			mapof.Any{"host": factory.Hostname()},
			queue.WithPriority(500),
		)

		// Schedule "Shuffle" tasks
		q.Enqueue <- queue.NewTask(
			"Shuffle",
			mapof.Any{"host": factory.Hostname()},
			queue.WithPriority(300),
		)

		// Schedule "PollFollowing" tasks every four hours, starting at 1am.
		if isHour(4, 1) {

			q.Enqueue <- queue.NewTask(
				"PollFollowing",
				mapof.Any{"host": factory.Hostname()},
				queue.WithPriority(500),
			)
		}
	}

	return queue.Success()
}
