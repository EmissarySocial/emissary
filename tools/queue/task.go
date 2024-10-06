package queue

import (
	"time"
)

// Task wraps a Task with the metadata required to track its runs and retries.
type Task struct {
	TaskID      string         `bson:"taskId"`      // Unique identfier for this task
	LockID      string         `bson:"lockId"`      // Unique identifier for the worker that is currently processing this task
	Name        string         `bson:"name"`        // Name of the task (used to identify the handler function)
	Arguments   map[string]any `bson:"arguments"`   // Data required to execute this task (marshalled as a map)
	CreateDate  int64          `bson:"createDate"`  // Unix epoch seconds when this task was created
	StartDate   int64          `bson:"startDate"`   // Unix epoch seconds when this task is scheduled to execute
	TimeoutDate int64          `bson:"timeoutDate"` // Unix epoch seconds when this task will "time out" and can be reclaimed by another process
	Priority    int            `bson:"priority"`    // Priority of the handler, determines the order that tasks are executed in.
	RetryCount  int            `bson:"retryCount"`  // Number of times that this task has already been retried
	RetryMax    int            `bson:"retryMax"`    // Maximum number of times that this task can be retried
	Error       error          `bson:"error"`       // Error (if any) from the last execution
}

// NewTask uses a Task object to create a new Task record
// that can be saved to a Storage provider.
func NewTask(name string, arguments map[string]any, options ...TaskOption) Task {

	now := time.Now().Unix()

	result := Task{
		TaskID:      "",
		LockID:      "",
		Name:        name,
		Arguments:   arguments,
		CreateDate:  now,
		StartDate:   now,
		TimeoutDate: 0,
		Priority:    16,
		RetryCount:  0,
		RetryMax:    12, // With exponential backoff, 2^12 minutes = 4096 minutes ~= 68 hours
	}

	// Apply functional options
	for _, option := range options {
		option(&result)
	}

	return result
}

// Delay sets the time.Duration before the task is executed
func (task *Task) Delay(delay time.Duration) {
	task.StartDate = time.Now().Add(delay).Unix()
}
