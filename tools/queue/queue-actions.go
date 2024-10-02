package queue

import (
	"time"

	"github.com/benpate/derp"
)

// taskSucceeded marks a task as completed, and removes it from the queue.
func (q *Queue) taskSucceeded(task Task) error {

	const location = "queue.taskSucceeded"

	// TODO: Save statistics here?

	// Remove the task from the queue
	if err := q.storage.DeleteTask(task); err != nil {
		return derp.Wrap(err, location, "Unable to remove task from queue")
	}

	// Silence is golden
	return nil
}

// taskError marks a task as errored and attempts to re-queue it for later.
// If the task has already been retried too many times, then it will be moved
// to the error log and removed from the queue.
func (q *Queue) taskError(task Task, err error) error {

	// If the task has already been (re)tried too many times, then give up
	// and move it to the error log
	if task.RetryCount >= task.RetryMax {
		return q.taskFailure(task, err)
	}

	// Update the task data and re-queue it
	task.WorkerID = ""
	task.Running = false
	task.StartDate = time.Now().Add(backoff(task.RetryCount)).Unix()
	task.TimeoutDate = 0
	task.RetryCount++
	task.Error = err

	return q.storage.SaveTask(task)
}

// taskFailure marks a task as failed and moves it to the error log.
func (q *Queue) taskFailure(task Task, err error) error {

	const location = "queue.taskFailure"

	// Update the task with the provided error
	task.Error = err

	// Add the task to the error log
	if err := q.storage.LogTask(task); err != nil {
		return derp.Wrap(err, location, "Unable to add task to error log")
	}

	if err := q.storage.DeleteTask(task); err != nil {
		return derp.Wrap(err, location, "Unable to remove task from queue")
	}

	return nil
}
