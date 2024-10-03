package queue

// Option is a function that modifies a Queue object
type Option func(*Queue)

// WithStorage sets the storage and unmarshaller for the queue
func WithStorage(storage Storage, unmarshaller Unmarshaller) Option {
	return func(q *Queue) {
		q.storage = storage
		q.unmarshaller = unmarshaller
	}
}

// WithWorkerCount sets the number of concurrent processes to run
func WithWorkerCount(workerCount int) Option {
	return func(q *Queue) {
		q.workerCount = workerCount
	}
}

// WithBufferSize sets the number of tasks to lock in a single transaction
func WithBufferSize(bufferSize int) Option {
	return func(q *Queue) {
		q.bufferSize = bufferSize
	}
}

// WithPollStorage sets whether the queue should poll the storage for new tasks
func WithPollStorage(pollStorage bool) Option {
	return func(q *Queue) {
		q.pollStorage = pollStorage
	}
}
