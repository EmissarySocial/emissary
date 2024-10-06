package queue

import "time"

type TaskOption func(*Task)

// WithPriority sets the priority of the task
func WithPriority(priority int) TaskOption {
	return func(t *Task) {
		t.Priority = priority
	}
}

// WithDelaySeconds sets the number of seconds before the task is executed
func WithDelaySeconds(delaySeconds int) TaskOption {
	return func(task *Task) {
		task.Delay(time.Duration(delaySeconds) * time.Second)
	}
}

// WithDelayMinutes sets the number of minutes before the task is executed
func WithDelayMinutes(delayMinutes int) TaskOption {
	return func(task *Task) {
		task.Delay(time.Duration(delayMinutes) * time.Minute)
	}
}

// WithDelayHours sets the number of hours before the task is executed
func WithDelayHours(delayHours int) TaskOption {
	return func(task *Task) {
		task.Delay(time.Duration(delayHours) * time.Hour)
	}
}

// WithRetryMax sets the maximum number of times that a task can be retried
func WithRetryMax(retryMax int) TaskOption {
	return func(t *Task) {
		t.RetryMax = retryMax
	}
}
