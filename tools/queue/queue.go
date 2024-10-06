package queue

import (
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/channel"
)

type Queue struct {
	storage     Storage       // Storage is the interface to the database
	marshallers []Marshaller  // Unmarshaller is the interface to convert Journal objects into Task objects
	workerCount int           // Number of goroutines to use for processing Tasks concurrently. Default process count is 16
	bufferSize  int           // BufferSize determines the number of Tasks to lock in one transaction. Default buffer size is 32
	pollStorage bool          // PollStorage determines if the queue should poll the database for new tasks. Default is true
	buffer      chan Journal  // TaskBuffer is a channel of tasks that are ready to be processed
	done        chan struct{} // Done channel is called to stop the queue
}

// NewQueue returns a fully initialized Queue object, with all options applied
func NewQueue(options ...Option) Queue {

	// Create the new Queue object
	result := Queue{
		workerCount: 16,
		bufferSize:  32,
		pollStorage: true,
	}

	// Apply options
	for _, option := range options {
		option(&result)
	}

	// Create the task buffer last (to use the correct buffer size)
	result.buffer = make(chan Journal, result.bufferSize)

	// Start `ProcessCount` processes to listen for new Tasks
	for range result.workerCount {
		go result.startWorker()
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

		if channel.Closed(q.done) {
			return
		}

		// Loop through any existing tasks that are locked by this worker
		journals, err := q.storage.GetTasks()

		if err != nil {
			derp.Report(err)
			continue
		}

		// If there are no tasks, wait one minute before trying to lock more.
		if len(journals) == 0 {
			time.Sleep(1 * time.Minute)
		}

		// Loop through all tasks that we have to process
		for _, journal := range journals {

			if channel.Closed(q.done) {
				return
			}

			q.buffer <- journal
		}
	}
}

// RunTask
func (q *Queue) Push(task Task) error {

	const location = "queue.Push"

	// Special Case #1: for immediate execution, just run the Task directly
	if task.Priority() == 0 {
		go func() {
			if err := task.Run(); err != nil {

				// If the task fails, then create a Journal and save to Storage provider
				journal := NewJournal(task, 0)

				if err := q.onTaskError(journal, err); err != nil {
					derp.Report(err)
				}
			}
		}()

		return nil
	}

	// Special Case #2: If there is no storage provider, queue the Task in the memory buffer
	if q.storage == nil {
		journal := NewJournal(task, 0)
		q.buffer <- journal
		return nil
	}

	// Otherwise, write the Task to the Storage provider
	journal := NewJournal(task, 0)

	if err := q.storage.SaveTask(journal); err != nil {
		return derp.Wrap(err, location, "Error saving task to database")
	}

	return nil
}

func (q *Queue) Schedule(task Task, delay time.Duration) error {

	const location = "queue.Schedule"

	if q.storage == nil {
		return derp.NewInternalError(location, "Must have a storage provider in order to schedule tasks")
	}

	// Create a Journal record to save to the Storage provider
	journal := NewJournal(task, delay)

	// Save the Journal to the Storage provider
	if err := q.storage.SaveTask(journal); err != nil {
		return derp.Wrap(err, location, "Error saving task to database")
	}

	return nil
}

// Stop closes the queue and stops all workers (after they complete their current task)
func (queue *Queue) Stop() {
	close(queue.done)
}
