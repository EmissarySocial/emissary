// Package postcommit implements the transaction-safe task publication gate described in
// emissary-specs/POST-COMMIT-TASKS-DESIGN.md: queue tasks requested while a database
// transaction is open are spooled (as plain queue.Task values) and published only after
// the transaction commits.  A rolled-back transaction publishes nothing.  Tasks requested
// outside a transaction are published immediately.
//
// Without this gate, a task published mid-transaction can be executed (on a separate,
// majority-read database session) before the transaction commits, so the worker cannot
// see rows the task's own arguments reference.
package postcommit

import (
	"context"
	"sync"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// contextKey is the private key type under which a Tasks spool travels in a context.Context.
type contextKey struct{}

// Tasks is a per-transaction spool of queue.Task values.  WithTransaction places it into
// the transaction's context, Publish fills it, and it is flushed to the queue only after
// the transaction commits.
type Tasks struct {
	mutex sync.Mutex
	tasks []queue.Task
}

// NewTasks returns a fully initialized (empty) task spool.
func NewTasks() *Tasks {
	return &Tasks{}
}

// WithContext returns a child context that carries the provided task spool.
func WithContext(ctx context.Context, tasks *Tasks) context.Context {
	return context.WithValue(ctx, contextKey{}, tasks)
}

// From returns the task spool carried by the provided context, or nil if there is none
// (i.e. no transaction is open).
func From(ctx context.Context) *Tasks {

	if tasks, ok := ctx.Value(contextKey{}).(*Tasks); ok {
		return tasks
	}

	return nil
}

// Add appends a task to the spool.  Safe for concurrent use.
func (t *Tasks) Add(task queue.Task) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tasks = append(t.tasks, task)
}

// Reset discards all spooled tasks.  It is called at the start of every transaction
// attempt: the mongo driver may run a transaction callback multiple times on transient
// errors, and only the successful attempt's tasks may survive.
func (t *Tasks) Reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tasks = nil
}

// Drain returns all spooled tasks in FIFO order and clears the spool.
func (t *Tasks) Drain() []queue.Task {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	result := t.tasks
	t.tasks = nil
	return result
}

// Publish is the one call that services make to emit a background task.  If the session
// is part of an open transaction (its context carries a Tasks spool), the task is spooled
// and published only after the transaction commits.  Otherwise (GET requests, schedulers,
// startup) the task is published to the queue immediately.  Matching queue.NewTask,
// publish errors are reported, never returned.
func Publish(session data.Session, q *queue.Queue, name string, args mapof.Any, options ...queue.TaskOption) {

	const location = "tools.postcommit.Publish"

	task := queue.NewTask(name, args, options...)

	// RULE: session must never be nil here — every production Builder carries a live
	// data.Session, so a nil one is a broken invariant (a Builder assembled without a session.
	if session == nil {

		// We can't inspect a transaction context without it, so report the defect
		// for observability and fall through to immediate publish: the safest available
		// action, since it preserves the task and never panics this fire-and-forget path.
		derp.Report(derp.Internal(location, "Nil session passed to Publish (this should never happen)", name))

	} else if spool := From(session.Context()); spool != nil {
		// Transactional context: spool for post-commit publication.
		spool.Add(task)
		return
	}

	// No queue configured (some test harnesses): preserve the no-op-safe behavior of the
	// call sites this function replaces.
	if q == nil {
		return
	}

	// Non-transactional context: publish immediately.
	if err := q.Publish(task); err != nil {
		derp.Report(derp.Wrap(err, location, "Publishing task", name))
	}
}

// WithTransaction wraps server.WithTransaction with the post-commit spool lifecycle: a
// fresh spool rides the transaction's context, is reset on every callback attempt, is
// dropped on rollback, and is published FIFO to the queue after a successful commit.
// Publish failures after commit are reported, never returned — the transaction is already
// durable, and the queue's own retry semantics take over from there.
func WithTransaction(ctx context.Context, server data.Server, q *queue.Queue, callback data.TransactionCallbackFunc) (any, error) {

	const location = "tools.postcommit.WithTransaction"

	spool := NewTasks()
	ctx = WithContext(ctx, spool)

	result, err := server.WithTransaction(ctx, func(session data.Session) (any, error) {
		spool.Reset() // the driver may retry the callback; only the winning attempt's tasks survive
		return callback(session)
	})

	// Rollback/error: spooled tasks are dropped, never published.
	if err != nil {
		return result, err
	}

	// COMMIT: publish the spool in FIFO order.
	if q != nil {
		for _, task := range spool.Drain() {
			if publishError := q.Publish(task); publishError != nil {
				derp.Report(derp.Wrap(publishError, location, "Publishing post-commit task", task.Name))
			}
		}
	}

	return result, nil
}
