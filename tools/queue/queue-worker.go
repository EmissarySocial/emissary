package queue

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/channel"
)

// startWorker runs a single worker process, pulling Tasks off
// the buffered channel and running them one at a time.
func (q *Queue) startWorker() {

	// Pull Tasks off of the buffereed channel
	for task := range q.buffer {

		// Execute the Task
		if err := q.consume(task); err != nil {
			derp.Report(err)
		}

		// If the queue has stopped, then exit the worker
		if channel.Closed(q.done) {
			return
		}
	}
}

// consume executes a single Task
func (q *Queue) consume(task Task) error {

	const location = "queue.processOne"

	consumer, exists := q.consumers[task.Name]

	// If the consumer does not exist, then this Task is invalid.
	// Marking it as a failure will remove it from the queue.
	if !exists {
		err := derp.NewInternalError(location, "Consumer does not exist", task.Name)
		if err := q.onTaskFailure(task, err); err != nil {
			derp.Report(err)
			return nil
		}
	}

	// Try to run the Task
	if runError := consumer.Run(task); runError != nil {

		// If the Task fails, then try to re-queue or handle the error
		if writeError := q.onTaskError(task, runError); writeError != nil {
			return derp.Wrap(writeError, location, "Error setting task error", runError)
		}

		// Report the error but do not return it because we have re-queued the task to try again
		// derp.Report(derp.Wrap(runError, location, "Error executing task"))
		return nil
	}

	// Otherwise, the Task was successful.  Remove it from the Storage provider
	if runError := q.onTaskSucceeded(task); runError != nil {
		return derp.Wrap(runError, location, "Error setting task success")
	}

	// Success! uwu
	return nil
}
