package queue

import "time"

// Journal wraps a Task with the metadata required to track its runs and retries.
type Journal struct {
	TaskID      string `bson:"taskId"`      // Unique identfier for this task
	LockID      string `bson:"lockId"`      // Unique identifier for the worker that is currently processing this task
	CreateDate  int64  `bson:"createDate"`  // Unix epoch seconds when this task was created
	StartDate   int64  `bson:"startDate"`   // Unix epoch seconds when this task is scheduled to execute
	TimeoutDate int64  `bson:"timeoutDate"` // Unix epoch seconds when this task will "time out" and can be reclaimed by another process
	Priority    int    `bson:"priority"`    // Priority of the handler, determines the order that tasks are executed in.
	RetryCount  int    `bson:"retryCount"`  // Number of times that this task has already been retried
	RetryMax    int    `bson:"retryMax"`    // Maximum number of times that this task can be retried

	Task      Task           `bson:"-"`         // Task object that this journal encapsulates
	Arguments map[string]any `bson:"arguments"` // Data required to execute this task (marshalled as a map)
	Error     error          `bson:"error"`     // Error (if any) from the last execution
}

// NewJournal uses a Task object to create a new Journal record
// that can be saved to a Storage provider.
func NewJournal(task Task, delay time.Duration) Journal {

	now := time.Now()

	return Journal{
		TaskID:      task.TaskID(),
		LockID:      "",
		CreateDate:  now.Unix(),
		StartDate:   now.Add(delay).Unix(),
		TimeoutDate: 0,
		Priority:    task.Priority(),
		RetryCount:  0,
		RetryMax:    task.RetryMax(),
		Task:        task,
		Arguments:   task.MarshalMap(),
	}
}
