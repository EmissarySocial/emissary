package queue

// Tasks represents a single operation that can be queued
// (and possibly serialized to a storage system)
type Task interface {

	// TaskID returns a unique identifier for this task
	TaskID() string

	// Priority returns the priority of this task
	// Priority = 0 is executed immediately
	// Priority > 0 is executed in ascending order
	Priority() int

	// RetryMax returns the number of times that this task can be retried
	RetryMax() int

	// Run executes the task, and returns an error if unsuccessul.
	// The queue is responsible for handling retries and timeouts.uwu
	Run() error

	// MarshalMap exports all of the Task data into a map[string]any
	MarshalMap() map[string]any
}
