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

		// Schedule "Shuffle" tasks
		q.NewTask(
			"Shuffle",
			mapof.Any{"host": factory.Hostname()},
		)

		// Schedule "PollFollowing-Index" tasks every four hours, starting at 1am.
		if isHour(4, 1) {

			q.NewTask(
				"PollFollowing-Index",
				mapof.Any{"host": factory.Hostname()},
			)
		}
	}

	return queue.Success()
}
