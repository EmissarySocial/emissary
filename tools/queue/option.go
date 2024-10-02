package queue

// Option is a function that modifies a Queue object
type Option func(*Queue)

// WithProcessCount sets the number of concurrent processes to run
func WithProcessCount(processCount int) Option {
	return func(q *Queue) {
		q.processCount = processCount
	}
}

// WithBufferSize sets the number of tasks to lock in a single transaction
func WithBufferSize(bufferSize int) Option {
	return func(q *Queue) {
		q.bufferSize = bufferSize
	}
}

// WithTimeout sets the default timeout for tasks, after which they will be retried
func WithTimeout(timeoutMinutes int) Option {
	return func(q *Queue) {
		q.timeoutMinutes = timeoutMinutes
	}
}

// WithHandler adds a new task handler to the queue
func WithHandler(name string, handler Handler) Option {
	return func(q *Queue) {
		q.handlers[name] = handler
	}
}
