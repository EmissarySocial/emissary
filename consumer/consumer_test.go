package consumer

import (
	"errors"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/service"
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

// TestConsumer_Run_ClaimsGeocode confirms that the name the geocode retry is published under is
// one the consumer claims, because an unclaimed task is failed and never run (BUG-242)
func TestConsumer_Run_ClaimsGeocode(t *testing.T) {

	// The task has no hostname, so it stops in WithFactory, which only a claimed task reaches
	result := New(nil).Run(queue.Task{Name: service.TaskGeocode, Arguments: mapof.NewAny()})
	require.Equal(t, queue.ResultStatusFailure, result.Status)

	// The name that was published until BUG-242 is claimed by no one
	result = New(nil).Run(queue.Task{Name: "GeocodeAddress", Arguments: mapof.NewAny()})
	require.Equal(t, queue.ResultStatusIgnored, result.Status)
}

// TestConsumer_LifecycleHooksAreSilent pins the DEFAULT of the three outcome hooks.  They run for
// every task in the system, and only SyncStreamSource acts on them today -- see
// syncStreamSource_test.go.  The nil ServerFactory is the proof: a hook that dispatched would
// reach for it.
func TestConsumer_LifecycleHooksAreSilent(t *testing.T) {

	consumer := New(nil)
	task := queue.Task{Name: "Geocode", Arguments: mapof.NewAny(), RetryCount: 3, RetryMax: 8}
	failure := errors.New("something went wrong")

	require.NoError(t, consumer.OnSuccess(task))
	require.NoError(t, consumer.OnError(task, failure))
	require.NoError(t, consumer.OnFailure(task, failure))
}

// TestConsumer_OnPublish_SetsPriority confirms that a task entering the queue takes its priority
// from the table in PreProcessor, and that a priority its publisher chose is kept
func TestConsumer_OnPublish_SetsPriority(t *testing.T) {

	consumer := New(nil)

	// A task with no priority takes the one listed for its name
	for name, expected := range map[string]int{
		"Geocode":                              64,
		service.StreamOutboxTaskName:           16,
		service.StreamFillFingerprintsTaskName: 256,
		"RecycleDomain":                        1024,
	} {
		task := queue.NewTask(name, mapof.NewAny())
		require.Equal(t, -1, task.Priority, "a new task starts with no priority")
		require.NoError(t, consumer.OnPublish(&task))
		require.Equal(t, expected, task.Priority, name)
	}

	// A priority that the publisher chose is kept
	chosen := queue.NewTask("Geocode", mapof.NewAny(), queue.WithPriority(8))
	require.NoError(t, consumer.OnPublish(&chosen))
	require.Equal(t, 8, chosen.Priority)

	// A name that the table does not list is left for the queue's default
	unknown := queue.NewTask("NotInTheTable", mapof.NewAny())
	require.NoError(t, consumer.OnPublish(&unknown))
	require.Equal(t, -1, unknown.Priority)
}

// TestConsumer_PrioritiesReachTheQueue confirms that a queue built with this Consumer stores a
// background task under the priority the table lists, not the queue's default
func TestConsumer_PrioritiesReachTheQueue(t *testing.T) {

	storage := &recordingStorage{}
	taskQueue := queue.New(queue.WithConsumers(New(nil)), queue.WithStorage(storage), queue.WithRunImmediatePriority(32))
	t.Cleanup(taskQueue.Stop)

	require.NoError(t, taskQueue.Publish(queue.NewTask("RecycleDomain", mapof.NewAny())))
	require.Equal(t, []int{1024}, storage.priorities())
}

// recordingStorage is a queue.Storage that keeps every task saved to it, and hands none back
type recordingStorage struct {
	mutex sync.Mutex
	saved []queue.Task
}

// GetTasks returns no tasks, so that a stored task stays where the test can read it
func (storage *recordingStorage) GetTasks() ([]queue.Task, error) {
	return nil, nil
}

// SaveTask records the task
func (storage *recordingStorage) SaveTask(task queue.Task) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	storage.saved = append(storage.saved, task)
	return nil
}

// DeleteTask does nothing, because no task is handed out
func (storage *recordingStorage) DeleteTask(string) error {
	return nil
}

// DeleteTaskBySignature does nothing, because no task is handed out
func (storage *recordingStorage) DeleteTaskBySignature(string) error {
	return nil
}

// LogFailure does nothing, because no task runs
func (storage *recordingStorage) LogFailure(queue.Task) error {
	return nil
}

// priorities returns the priority of every saved task, in order
func (storage *recordingStorage) priorities() []int {

	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	result := make([]int, 0, len(storage.saved))

	for _, task := range storage.saved {
		result = append(result, task.Priority)
	}

	return result
}
