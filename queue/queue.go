/*
Package queue implements a simple queue for passing asynchronous
tasks to a pool of workers.  This will probably get expanded or
replaced in the future by a more robust queueing system.
*/
package queue

import (
	"github.com/benpate/derp"
)

type Queue struct {
	tasks chan Task
	close chan bool
}

// NewQueue creates a new queue (channel).  You can specify the maximum length
// of the buffer, along with the number of workers that will pull tasks off of the queue.
func NewQueue(length int, workers int) *Queue {
	result := &Queue{
		tasks: make(chan Task, length),
		close: make(chan bool),
	}

	for i := 0; i < workers; i++ {
		go result.Worker()
	}

	return result
}

// Worker runs a single worker, which runs tasks from the queue sequentially.
func (q *Queue) Worker() {
	for {
		select {
		case task := <-q.tasks:
			if err := task.Run(); err != nil {
				derp.Report(derp.Wrap(err, "queue.Queue.Worker", "Error running task", task))
			}
		case <-q.close:
			return
		}
	}
}

// Run adds a task to the queue, to be run asynchronously when it is possible.
func (q *Queue) Run(task Task) {
	go func() {
		q.tasks <- task
	}()
}

// CLose closes the queue and stops the workers.
func (q *Queue) Close() {
	q.close <- true
}
