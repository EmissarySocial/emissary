package queue

import (
	"github.com/benpate/derp"
)

// startProcessor runs a single queue process, pulling Tasks off of the queue channel and executing them
func (q *Queue) startProcessor() {

	for task := range q.taskBuffer {
		if err := q.processOne(task); err != nil {
			derp.Report(err)
		}
	}
}

// processOne executes a single task
func (q *Queue) processOne(task Task) error {

	const location = "queue.processOne"

	// Locate the handler for this task
	handler, handlerExists := q.handlers[task.Name]

	if !handlerExists {
		handleError := derp.NewInternalError(location, "No handler defined for task name: "+task.Name)
		if writeError := q.taskFailure(task, handleError); writeError != nil {
			return derp.Wrap(writeError, location, "Error setting task failed", handleError)
		}

		// Return this error because the queue is misconfigured
		return handleError
	}

	// Execute the handler function, log errors
	if handleError := handler(task.Arguments); handleError != nil {
		if writeError := q.taskError(task, handleError); writeError != nil {
			return derp.Wrap(writeError, location, "Error setting task error", handleError)
		}

		// Report the error - but do not return it because we have re-queued the task to try again
		derp.Report(derp.Wrap(handleError, location, "Error executing task"))
		return nil
	}

	// Fall through means success.  Remove the task from the queue
	if err := q.taskSucceeded(task); err != nil {
		return derp.Wrap(err, location, "Error setting task success")
	}

	return nil
}
