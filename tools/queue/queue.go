package queue

import (
	"time"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Queue struct {
	storage        Storage            // Storage is the interface to the database
	handlers       map[string]Handler // Handlers is a map of functions that can be called by the queue
	timeoutMinutes int                // Default task timeout is 30 minutes
	processCount   int                // Number of goroutines to use for processing Tasks concurrently. Default process count is 16
	bufferSize     int                // BufferSize determines the number of Tasks to lock in one transaction. Default buffer size is 32
	taskBuffer     chan Task          // TaskBuffer is a channel of tasks that are ready to be processed
	pollStorage    bool               // PollStorage determines if the queue should poll the database for new tasks. Default is true
}

// NewQueue returns a fully initialized Queue object, with all options applied
func NewQueue(options ...Option) Queue {

	// Create the new Queue object
	result := Queue{
		storage:        nil,
		processCount:   16,
		bufferSize:     32,
		timeoutMinutes: 30,
		pollStorage:    true,
	}

	// Apply options
	for _, option := range options {
		option(&result)
	}

	// Create the task buffer last (to use the correct buffer size)
	result.taskBuffer = make(chan Task, result.bufferSize)

	// Start `ProcessCount` processes to listen for new Tasks
	for range result.processCount {
		go result.startProcessor()
	}

	// Poll the storage container for new Tasks
	go result.start()

	// UwU LOL.
	return result
}

// Start runs the queue and listens for new tasks
func (q *Queue) start() {

	// If we don't have a storage object, then we won't poll it for update
	if q.storage == nil {
		return
	}

	// If this service is not configured to poll the database, then return
	if !q.pollStorage {
		return
	}

	// Poll the storage container for new Tasks
	for {
		lockID := primitive.NewObjectID()

		// Try to lock more tasks if we don't already have any
		if err := q.storage.LockTasks(lockID); err != nil {
			derp.Report(err)
			continue
		}

		// Loop through any existing tasks that are locked by this worker
		tasks, err := q.storage.GetTasks(lockID)

		if err != nil {
			derp.Report(err)
			continue
		}

		// If there are no tasks, wait one minute before trying to lock more.
		if len(tasks) == 0 {
			time.Sleep(1 * time.Minute)
		}

		// Loop through all tasks that we have to process
		for _, task := range tasks {
			q.taskBuffer <- task
		}
	}
}

// RunTask
func (q *Queue) Push(task Task) error {

	const location = "queue.Enqueue"

	// If requested, try to process Tasks immediately without hitting the queue
	if task.TryBeforeQueue {
		if err := q.processOne(task); err != nil {
			return derp.Wrap(err, location, "Error processing task before queueing")
		}

		return nil
	}

	// Reset Task values (just in case)
	task.WorkerID = ""
	task.TimeoutDate = 0
	task.RetryCount = 0
	task.Running = false

	// Fall through means we're adding the task to the queue database
	if err := q.storage.SaveTask(task); err != nil {
		return derp.Wrap(err, location, "Error saving task to database")
	}

	return nil
}
