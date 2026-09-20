package consumer

import (
	"errors"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestConsumer_IgnoresUnknownTasks confirms that a task Emissary does not own is passed back to
// the queue.  The nil ServerFactory is the proof: a task that was recognized would reach for it.
func TestConsumer_IgnoresUnknownTasks(t *testing.T) {

	consumer := New(nil)

	for _, name := range []string{"", "NotATask", "geocode", "Geocode-Extra"} {
		result := consumer.Run(queue.Task{Name: name, Arguments: mapof.NewAny()})
		require.Equal(t, queue.ResultStatusIgnored, result.Status, "task %q", name)
	}
}

// TestConsumer_LifecycleHooksAreSilent pins the current state of the four lifecycle hooks:
// declared because queue.Consumer requires them, and reporting nothing yet.
func TestConsumer_LifecycleHooksAreSilent(t *testing.T) {

	consumer := New(nil)
	task := queue.Task{Name: "Geocode", Arguments: mapof.NewAny(), RetryCount: 3, RetryMax: 8}
	failure := errors.New("something went wrong")

	require.NoError(t, consumer.OnPublish(&task))
	require.NoError(t, consumer.OnSuccess(task))
	require.NoError(t, consumer.OnError(task, failure))
	require.NoError(t, consumer.OnFailure(task, failure))
}
