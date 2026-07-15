package consumer

import (
	"github.com/benpate/turbine/queue"
)

// ScheduleStartup queues the one-time tasks that run when the server starts.
func ScheduleStartup(serverFactory ServerFactory) queue.Result {

	// There are no startup tasks right now. The hook stays wired up (consumer/schedule.go publishes
	// it on every boot) because it is where a one-time migration goes when one is next needed.

	// Stupendous.
	return queue.Success()
}
