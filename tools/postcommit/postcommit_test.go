package postcommit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Test Doubles
 ******************************************/

// testSession is a minimal data.Session whose only meaningful behavior is carrying a context.
type testSession struct {
	ctx context.Context
}

// Collection implements the data.Session interface. The stub owns no collections.
func (s testSession) Collection(_ string) data.Collection { return nil }

// Context implements the data.Session interface, returning the carried context
func (s testSession) Context() context.Context { return s.ctx }

// Close implements the data.Session interface. The stub holds no resources to release.
func (s testSession) Close() {}

// testServer is a data.Server that runs the transaction callback `attempts` times
// (simulating mongo driver retries on transient errors) and then returns `err`
// (simulating a rollback when non-nil).
type testServer struct {
	attempts int
	err      error
}

// Session implements the data.Server interface, returning a session that carries the provided context
func (s testServer) Session(ctx context.Context) (data.Session, error) {
	return testSession{ctx: ctx}, nil
}

// WithTransaction implements the data.Server interface, replaying the callback to simulate driver retries
func (s testServer) WithTransaction(ctx context.Context, fn data.TransactionCallbackFunc) (any, error) {

	var result any
	var err error

	attempts := max(s.attempts, 1)

	for range attempts {
		result, err = fn(testSession{ctx: ctx})

		if err != nil {
			return result, err
		}
	}

	return result, s.err
}

// newTestQueue returns a single-worker turbine queue whose consumer forwards every task
// name to the returned channel, preserving publish order.
func newTestQueue() (*queue.Queue, chan string) {

	received := make(chan string, 32)

	q := queue.New(
		queue.WithWorkerCount(1),
		queue.WithConsumers(func(name string, _ map[string]any) queue.Result {
			received <- name
			return queue.Success()
		}),
	)

	q.Start()
	return q, received
}

// expectTask waits for a single task name to arrive, failing the test on timeout.
func expectTask(t *testing.T, received chan string) string {
	t.Helper()

	select {
	case name := <-received:
		return name
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a published task")
		return ""
	}
}

// expectNoTask asserts that nothing arrives within a short window.
func expectNoTask(t *testing.T, received chan string) {
	t.Helper()

	select {
	case name := <-received:
		t.Fatalf("expected no published tasks, but received %q", name)
	case <-time.After(100 * time.Millisecond):
	}
}

/******************************************
 * Spool Primitives
 ******************************************/

// TestTasks_AddDrain_FIFO verifies that Drain returns spooled tasks in insertion order, then empties the spool
func TestTasks_AddDrain_FIFO(t *testing.T) {

	spool := postcommit.NewTasks()
	spool.Add(queue.NewTask("first", mapof.Any{}))
	spool.Add(queue.NewTask("second", mapof.Any{}))
	spool.Add(queue.NewTask("third", mapof.Any{}))

	drained := spool.Drain()
	require.Len(t, drained, 3)
	require.Equal(t, "first", drained[0].Name)
	require.Equal(t, "second", drained[1].Name)
	require.Equal(t, "third", drained[2].Name)

	// Drain clears the spool
	require.Empty(t, spool.Drain())
}

// TestTasks_Reset verifies that Reset discards every spooled task
func TestTasks_Reset(t *testing.T) {

	spool := postcommit.NewTasks()
	spool.Add(queue.NewTask("doomed", mapof.Any{}))
	spool.Reset()

	require.Empty(t, spool.Drain())
}

// TestTasks_ConcurrentAdd verifies that the spool is safe for concurrent writers
func TestTasks_ConcurrentAdd(t *testing.T) {

	spool := postcommit.NewTasks()

	var group sync.WaitGroup
	for range 32 {
		group.Add(1)
		go func() {
			defer group.Done()
			spool.Add(queue.NewTask("concurrent", mapof.Any{}))
		}()
	}
	group.Wait()

	require.Len(t, spool.Drain(), 32)
}

// TestFrom_MissingSpool verifies that a context with no spool returns nil instead of panicking
func TestFrom_MissingSpool(t *testing.T) {
	require.Nil(t, postcommit.From(context.Background()))
}

/******************************************
 * Publish
 ******************************************/

// TestPublish_SpoolsInsideTransaction verifies that a task raised inside a transaction waits in the spool
func TestPublish_SpoolsInsideTransaction(t *testing.T) {

	q, received := newTestQueue()
	defer q.Stop()

	spool := postcommit.NewTasks()
	session := testSession{ctx: postcommit.WithContext(context.Background(), spool)}

	postcommit.Publish(session, q, "spooled-task", mapof.Any{})

	// The task is spooled, not published...
	expectNoTask(t, received)

	// ...and sits in the spool awaiting the commit.
	drained := spool.Drain()
	require.Len(t, drained, 1)
	require.Equal(t, "spooled-task", drained[0].Name)
}

// TestPublish_ImmediateOutsideTransaction verifies that a task raised outside a transaction publishes right away
func TestPublish_ImmediateOutsideTransaction(t *testing.T) {

	q, received := newTestQueue()
	defer q.Stop()

	session := testSession{ctx: context.Background()}
	postcommit.Publish(session, q, "immediate-task", mapof.Any{})

	require.Equal(t, "immediate-task", expectTask(t, received))
}

// TestPublish_NilQueueOutsideTransaction verifies that a nil queue drops the task instead of panicking
func TestPublish_NilQueueOutsideTransaction(t *testing.T) {

	// Must not panic (preserves the no-op-safe behavior of nil-queue call sites)
	session := testSession{ctx: context.Background()}
	postcommit.Publish(session, nil, "dropped-task", mapof.Any{})
}

/******************************************
 * WithTransaction
 ******************************************/

// TestWithTransaction_PublishesAfterCommit verifies that spooled tasks publish in order once the transaction commits
func TestWithTransaction_PublishesAfterCommit(t *testing.T) {

	q, received := newTestQueue()
	defer q.Stop()

	server := testServer{attempts: 1}

	result, err := postcommit.WithTransaction(context.Background(), server, q, func(session data.Session) (any, error) {
		postcommit.Publish(session, q, "one", mapof.Any{})
		postcommit.Publish(session, q, "two", mapof.Any{})

		// Nothing publishes while the transaction is open
		expectNoTask(t, received)
		return "done", nil
	})

	require.NoError(t, err)
	require.Equal(t, "done", result)

	// Published FIFO after commit
	require.Equal(t, "one", expectTask(t, received))
	require.Equal(t, "two", expectTask(t, received))
}

// TestWithTransaction_DropsTasksOnRollback verifies that a rolled-back transaction publishes nothing
func TestWithTransaction_DropsTasksOnRollback(t *testing.T) {

	q, received := newTestQueue()
	defer q.Stop()

	server := testServer{attempts: 1}
	rollback := errors.New("rollback")

	_, err := postcommit.WithTransaction(context.Background(), server, q, func(session data.Session) (any, error) {
		postcommit.Publish(session, q, "never-published", mapof.Any{})
		return nil, rollback
	})

	require.ErrorIs(t, err, rollback)
	expectNoTask(t, received)
}

// TestWithTransaction_ResetsSpoolOnRetry verifies that a retried transaction publishes each task exactly once
func TestWithTransaction_ResetsSpoolOnRetry(t *testing.T) {

	q, received := newTestQueue()
	defer q.Stop()

	// The server runs the callback THREE times (simulating driver retries on
	// TransientTransactionError).  Only the final attempt's task may publish.
	server := testServer{attempts: 3}

	_, err := postcommit.WithTransaction(context.Background(), server, q, func(session data.Session) (any, error) {
		postcommit.Publish(session, q, "retried-task", mapof.Any{})
		return nil, nil
	})

	require.NoError(t, err)

	require.Equal(t, "retried-task", expectTask(t, received))
	expectNoTask(t, received) // exactly one, not three
}
