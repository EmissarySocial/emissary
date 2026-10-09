package consumer

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// ScheduleStartup queues the one-time tasks that run when the server starts.
func ScheduleStartup(serverFactory ServerFactory) queue.Result {

	const location = "consumer.ScheduleStartup"

	// Repair every Stripe Connect Connection whose webhook secret was never stored.  This runs on
	// every boot, and does nothing once each Connection holds its secret (see AGENTS.md).
	for factory := range serverFactory.RangeDomains() {

		if !factory.Connection().StripeConnectNeedsRepair() {
			continue
		}

		// RULE: Signed per domain.  Every server runs ScheduleStartup, and two repairs at once
		// would each register a webhook endpoint.
		hostname := factory.Hostname()

		task := queue.NewTask(
			"RepairStripeConnect",
			mapof.Any{"hostname": hostname},
			queue.WithSignature("RepairStripeConnect:"+hostname),
		)

		if err := serverFactory.Queue().Publish(task); err != nil {
			derp.Report(derp.Wrap(err, location, "Queueing Stripe Connect repair", hostname))
		}
	}

	// Stupendous.
	return queue.Success()
}
