package queue

// Storage is the interface for persisting Tasks outside of memory
type Storage interface {

	// GetTasks retrieves a batch of Tasks from the Storage provider
	GetTasks() ([]Task, error)

	// SaveTask saves a Task to the Storage provider
	SaveTask(task Task) error

	// DeleteTask removes a Task from the Storage provider
	DeleteTask(taskID string) error

	// LogFailure writes a Task to the error log
	LogFailure(task Task) error
}
