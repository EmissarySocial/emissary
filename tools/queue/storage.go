package queue

import "go.mongodb.org/mongo-driver/bson/primitive"

// Storage is the interface for persisting Tasks outside of memory
type Storage interface {

	// LockTasks reserves a set of tasks to be processed by this worker / lockID
	LockTasks(primitive.ObjectID) error

	// GetTasks retrieves the tasks reserves for this worker / lockID
	GetTasks(primitive.ObjectID) ([]Task, error)

	// SaveTask saves a task to the storage
	SaveTask(Task) error

	// DeleteTask removes a task from the storage
	DeleteTask(Task) error

	// LogTask adds a task to the error log
	LogTask(Task) error
}
