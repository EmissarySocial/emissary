package queue

import (
	"time"

	"github.com/benpate/derp"
)

// onTaskSucceeded marks a task as completed, and removes it from the queue.
func (q *Queue) onTaskSucceeded(task Task) error {

	const location = "queue.onTaskSucceeded"

	// If there is no storage provider, then there's no stored record to remove.
	if q.storage == nil {
		return nil
	}

	// TODO: Calculate statistics here?

	// Remove the task from the queue
	if err := q.storage.DeleteTask(task.TaskID); err != nil {
		return derp.Wrap(err, location, "Unable to remove task from queue")
	}

	// Silence is golden
	return nil
}

// onTaskError marks a task as errored and attempts to re-queue it for later.
// If the task has already been retried too many times, then it will be moved
// to the error log and removed from the queue.
func (q *Queue) onTaskError(task Task, err error) error {

	// If the task has already been (re)tried too many times, then give up
	// and move it to the error log
	if task.RetryCount >= task.RetryMax {
		return q.onTaskFailure(task, err)
	}

	// Update the task data and re-queue it
	task.LockID = ""
	task.StartDate = time.Now().Add(backoff(task.RetryCount)).Unix()
	task.TimeoutDate = 0
	task.RetryCount++
	task.Error = err

	// If there is no storage provider, then use the buffer to queue the task
	if q.storage == nil {
		q.buffer <- task
		return nil
	}

	// Otherwise, write the Task back to the storage provider
	return q.storage.SaveTask(task)
}

// onTaskFailure marks a task as failed and moves it to the error log.
func (q *Queue) onTaskFailure(task Task, err error) error {

	const location = "queue.onTaskFailure"

	// Add the error into the Task record
	task.Error = err

	// If there is no storage provider, then there's not much we can do...
	// Just report the error and return
	if q.storage == nil {
		derp.Report(task.Error)
		return nil
	}

	// Add the task to the error log
	if err := q.storage.LogFailure(task); err != nil {
		return derp.Wrap(err, location, "Unable to add task to error log")
	}

	// Remove the task from the queue
	if err := q.storage.DeleteTask(task.TaskID); err != nil {
		return derp.Wrap(err, location, "Unable to remove task from queue")
	}

	// Succeeded in logging the failure, even if the Task itself failed.
	return nil
}
