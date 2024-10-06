package queue

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/channel"
)

// startWorker runs a single worker process, pulling Tasks off
// the buffered channel and running them one at a time.
func (q *Queue) startWorker() {

	// Pull Tasks off of the buffereed channel
	for journal := range q.buffer {

		// Execute the Task
		if err := q.runSingleTask(journal); err != nil {
			derp.Report(err)
		}

		// If the queue has stopped, then exit the worker
		if channel.Closed(q.done) {
			return
		}
	}
}

// runSingleTask executes a single Task
func (q *Queue) runSingleTask(journal Journal) error {

	const location = "queue.processOne"

	// If the Task has not already been unmarshalled from the Journal, then do it now
	if ok := q.unmarshal(&journal); !ok {
		journal.Error = derp.NewInternalError(location, "Unable to unmarshal Task")
		return q.storage.LogFailure(journal)
	}

	// Try to run the Task
	if runError := journal.Task.Run(); runError != nil {

		// If the Task fails, then try to re-queue or handle the error
		if writeError := q.onTaskError(journal, runError); writeError != nil {
			return derp.Wrap(writeError, location, "Error setting task error", runError)
		}

		// Report the error but do not return it because we have re-queued the task to try again
		// derp.Report(derp.Wrap(runError, location, "Error executing task"))
		return nil
	}

	// Otherwise, the Task was successful.  Remove it from the Storage provider
	if runError := q.onTaskSucceeded(journal.TaskID); runError != nil {
		return derp.Wrap(runError, location, "Error setting task success")
	}

	// Success! uwu
	return nil
}

// unmarshall attempts to unpack a Journal's map[string]any arguments into a a Task object.
// It returns TRUE if successful, and FALSE if the Task could not be unmarshalled.
func (q Queue) unmarshal(journal *Journal) bool {

	// If the task has already been unmarshalled, then don't try again.
	if journal.Task != nil {
		return true
	}

	// Try each marshaller in order, to try to unmarshall
	for _, marshaller := range q.marshallers {
		if marshaller.Unmarshal(journal) {
			return true
		}
	}

	// Fall through means that no marshaller was able to create a Task object
	return false
}
