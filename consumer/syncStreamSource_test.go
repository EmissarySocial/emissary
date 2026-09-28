package consumer

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/benpate/data"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

// unresolvableFactory is a ServerFactory that knows no domains.  A hook that tries to use it fails
// loudly, which is how these tests tell a hook that DISPATCHED from one that quietly returned nil.
type unresolvableFactory struct{}

// RangeDomains implements the ServerFactory interface, walking no domains
func (unresolvableFactory) RangeDomains() iter.Seq[*service.Factory] {
	return func(func(*service.Factory) bool) {}
}

// ByHostname implements the ServerFactory interface, resolving nothing
func (unresolvableFactory) ByHostname(string) (*service.Factory, error) {
	return nil, derp.NotFound("test", "No such domain")
}

// Queue implements the ServerFactory interface
func (unresolvableFactory) Queue() *queue.Queue { return nil }

// CommonDatabase implements the ServerFactory interface
func (unresolvableFactory) CommonDatabase() *mongo.Database { return nil }

// AllowPrivateIPs implements the ServerFactory interface
func (unresolvableFactory) AllowPrivateIPs() bool { return false }

// emptySession is a data.Session that holds nothing.  The malformed-ID guard returns before
// anything reaches it, but a literal nil is a shape that nilaway cannot prove safe.
type emptySession struct{}

// Collection implements the data.Session interface.  It panics rather than returning nil, so that
// a guard which stops running first fails loudly instead of nil-dereferencing somewhere else.
func (emptySession) Collection(string) data.Collection {
	panic("emptySession holds no collections")
}

// Context implements the data.Session interface
func (emptySession) Context() context.Context { return context.Background() }

// Close implements the data.Session interface.  The stub holds no resources to release.
func (emptySession) Close() {}

// syncTask returns a well-formed SyncStreamSource task
func syncTask() queue.Task {
	return queue.Task{
		Name: service.TaskSyncStreamSource,
		Arguments: mapof.Any{
			"hostname":       "example.com",
			"streamSourceId": "62f8f1d4d7f1c8e6b4a1c2d3",
		},
	}
}

// TestSyncStreamSource_MalformedID confirms that an unparseable identifier is a permanent failure.
// No retry can repair it, and a retryable Error would leave the task looping in the queue forever.
func TestSyncStreamSource_MalformedID(t *testing.T) {

	for _, value := range []string{"", "not-an-id", "62f8f1d4d7f1c8e6b4a1c2", "zzzzzzzzzzzzzzzzzzzzzzzz"} {

		// The nil factory is the proof that the guard runs before anything is loaded
		result := SyncStreamSource(nil, emptySession{}, mapof.Any{"streamSourceId": value})

		require.Equal(t, queue.ResultStatusFailure, result.Status, "value %q", value)
	}
}

// TestSyncStreamSource_IsRegistered confirms that the task reaches a handler.  A name that falls
// through the switch returns Ignored, and the task is silently dropped.
func TestSyncStreamSource_IsRegistered(t *testing.T) {

	result := New(unresolvableFactory{}).Run(syncTask())

	require.NotEqual(t, queue.ResultStatusIgnored, result.Status, "the task reaches no handler")
	require.Equal(t, queue.ResultStatusError, result.Status, "an unknown hostname may become known later")
}

// TestSyncStreamSource_Priority confirms that every synchronization, from Sync Now or a webhook,
// runs in the band that may run immediately.
func TestSyncStreamSource_Priority(t *testing.T) {

	task := syncTask()
	task.Priority = -1

	require.NoError(t, PreProcessor(&task))

	// RULE: Assert the EXACT value.  A name that matches no case leaves Priority at the -1
	// sentinel, and every "is it low enough?" comparison passes vacuously against that.
	require.Equal(t, 16, task.Priority)
}

// TestSyncStreamSource_HooksDispatch confirms that all three lifecycle hooks act on this task.
// Under D10 they are the only place a Status can be written: the task's own transaction has already
// rolled back by the time a failure is known, so a status written inside it would not survive.
func TestSyncStreamSource_HooksDispatch(t *testing.T) {

	consumer := New(unresolvableFactory{})
	failure := errors.New("something went wrong")

	task := syncTask()

	require.Error(t, consumer.OnSuccess(task), "OnSuccess must write SUCCESS")
	require.Error(t, consumer.OnError(task, failure), "OnError must record the retry message")
	require.Error(t, consumer.OnFailure(task, failure), "OnFailure must write FAILURE")
}

// TestSyncStreamSource_HooksIgnoreOtherTasks confirms that the hooks fire for THIS task and no
// other.  They run for every task in the system, so a missing name check would send every
// success in Emissary looking for a StreamSource record.
func TestSyncStreamSource_HooksIgnoreOtherTasks(t *testing.T) {

	consumer := New(unresolvableFactory{})
	failure := errors.New("something went wrong")

	for _, name := range []string{"Geocode", "DeleteStream", "", "SyncStreamSource-Extra", "SyncStreamSourceNow"} {

		task := syncTask()
		task.Name = name

		require.NoError(t, consumer.OnSuccess(task), "task %q", name)
		require.NoError(t, consumer.OnError(task, failure), "task %q", name)
		require.NoError(t, consumer.OnFailure(task, failure), "task %q", name)
	}
}
