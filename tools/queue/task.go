package queue

import (
	"github.com/benpate/rosetta/mapof"
)

// Task represents a single operation that the queue should perform
type Task struct {
	TaskID         string    // Unique identfier for this task
	LockID         string    // Unique identifier for the worker that is currently processing this task
	Name           string    // Task name / handler function to call to complete this
	Priority       int       // Priority of this task (higher is more important)
	WorkerID       string    // Name of the worker/server that is executing this task
	Arguments      mapof.Any // Arguments to pass to the task handler
	CreateDate     int64     // Unix epoch seconds when this task was created
	StartDate      int64     // Unix epoch seconds when this task is scheduled to execute
	TimeoutDate    int64     // Unix epoch seconds when this task will "time out" and can be reclaimed by another process
	Error          error     // Error (if any) from the last execution
	RetryMax       int       // Maximum number of times to retry this task before quitting. If 0 then do not retry at all.
	RetryCount     int       // Number of times that this task has already been retried
	Running        bool      // True if this task is currently being executed
	TryBeforeQueue bool      // True if this task should be tried once in place before writing to the queue
}

// NewTask create a fully initialized Task object
func NewTask() Task {
	return Task{
		Arguments: make(mapof.Any),
	}
}
