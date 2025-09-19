package consumer

import (
	"github.com/benpate/turbine/queue"
)

func ScheduleStartup(serverFactory ServerFactory) queue.Result {

	/*
		const location = "consumer.ScheduleStartup"
		log.Trace().Str("location", location).Msg("Running Startup Tasks...")

		time.Sleep(5 * time.Second) // Give the server a few seconds to finish starting up
		enqueue := serverFactory.Queue().Enqueue

		// Throughput test. Do not check in.
		{
			for i := range 6000 {
				enqueue <- queue.NewTask(
					"TestThroughput",
					mapof.Any{"value": i},
				)
			}
		}*/

	// Stupendous.
	return queue.Success()
}
