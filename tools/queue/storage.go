package queue

// Storage is the interface for persisting Tasks outside of memory
type Storage interface {

	// GetTasks retrieves a batch of Tasks from the Storage provider
	GetTasks() ([]Journal, error)

	// SaveTask saves a Task to the Storage provider
	SaveTask(journal Journal) error

	// DeleteTask removes a Task from the Storage provider
	DeleteTask(taskID string) error

	// LogFailure writes a Task to the error log
	LogFailure(journal Journal) error
}
