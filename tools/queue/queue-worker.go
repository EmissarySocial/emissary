package queue

import (
	"context"
	"time"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

func (q *Queue) StartWorker() {

}

func (q *Queue) LockTasks() ([]Task, error) {

	const location = "queue.Queue.LockTasks"

	// Look for tasks that are unassigned, or have expired and should be retried
	filter := bson.M{
		"$or": []bson.M{
			{"workerId": ""},
			{"retryDate": bson.M{"$lt": time.Now()}},
		},
	}

	// Lock selected tasks for this worker and reset their retry date
	update := bson.M{
		"$set": bson.M{
			"workerId":  q.WorkerID,
			"retryDate": time.Now().Add(time.Minute * time.Duration(q.TimeoutMinutes)),
		},
	}

	if _, err := q.Collection.UpdateMany(context.TODO(), filter, update); err != nil {
		return nil, derp.Wrap(err, location, "Error updating tasks")
	}

	return nil, nil
}
