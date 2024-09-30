package queue

import (
	"os"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/mongo"
)

type Queue struct {
	Collection     *mongo.Collection  // Collection is the MongoDB collection where tasks are stored
	Handlers       map[string]Handler // Handlers is a map of functions that can be called by the queue
	WorkerID       string             // WorkerID is the hostname of the server
	ProcessCount   int                // Default process count is 8
	TimeoutMinutes int                // Default task timeout is 30 minutes
}

func NewQueue(collection *mongo.Collection, options ...Option) (Queue, error) {

	// Find the server worker name
	workerID, err := os.Hostname()

	if err != nil {
		return Queue{}, derp.Wrap(err, "queue.NewQueue", "Error getting hostname")
	}

	// Create the new Queue object
	result := Queue{
		Collection:     collection,
		WorkerID:       workerID,
		ProcessCount:   8,
		TimeoutMinutes: 30,
	}

	// Apply options
	for _, option := range options {
		option(&result)
	}

	return result, nil
}
