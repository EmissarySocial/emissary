package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestDelivery_IsRegistered confirms that each delivery task reaches a handler. A name that falls
// through the switch returns Ignored, and the queue then fails the task as unrecognized.
func TestDelivery_IsRegistered(t *testing.T) {

	for _, name := range []string{delivery.TaskFanOut, delivery.TaskSend, service.TaskOutbox2Email} {

		task := queue.NewTask(name, mapof.Any{"hostname": "unknown.example"})
		result := New(unresolvableFactory{}).Run(task)

		require.NotEqual(t, queue.ResultStatusIgnored, result.Status, name)
		require.Equal(t, queue.ResultStatusError, result.Status, "%s: an unknown hostname may become known later", name)
	}
}

// TestDelivery_HostnameFromActor confirms that a send task, which carries no hostname argument,
// is routed by the host of the actor that sent the activity
func TestDelivery_HostnameFromActor(t *testing.T) {

	task := queue.NewTask(delivery.TaskSend, mapof.Any{"actor": "https://unknown.example/@user", "recipient": "https://remote.example/@alice", "body": "{}"})
	result := New(unresolvableFactory{}).Run(task)

	require.Equal(t, queue.ResultStatusError, result.Status, "the domain was looked up, and is not loaded")
}

// TestDelivery_Priorities confirms that a fan-out runs promptly after its commit, and that
// deliveries run below user-facing work
func TestDelivery_Priorities(t *testing.T) {

	// RULE: Assert the EXACT value.  A name that matches no case leaves Priority at the -1 sentinel.
	for name, expected := range map[string]int{
		delivery.TaskFanOut:      16,
		delivery.TaskSend:        256,
		service.TaskOutbox2Email: 256,
	} {
		task := queue.NewTask(name, nil)
		task.Priority = -1

		require.NoError(t, PreProcessor(&task))
		require.Equal(t, expected, task.Priority, name)
	}
}
