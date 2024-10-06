package queue_mongo

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/tools/queue"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage implements a queue Storage interface using MongoDB
type Storage struct {
	database       *mongo.Database // The mongodb database to read/write
	lockQuantity   int             // The number of tasks to lock at a time
	timeoutMinutes int             // Number of minutes to lock tasks before they are considered "timed out"
}

// New returns a fully initialized Storage object
func New(database *mongo.Database, lockQuantity int, timeoutMinutes int) Storage {
	return Storage{
		database:       database,
		lockQuantity:   lockQuantity,
		timeoutMinutes: timeoutMinutes,
	}
}

// SaveTask adds/updates a task to the queue
func (storage Storage) SaveTask(task queue.Task) error {

	const location = "queue.saveTask"
	timeout, cancel := timeoutContext(16)
	defer cancel()

	var taskID primitive.ObjectID
	var err error

	// If the Task does not have a TaskID, then create a new one
	if task.TaskID == "" {
		taskID = primitive.NewObjectID()
		task.TaskID = taskID.Hex()

	} else {

		taskID, err = primitive.ObjectIDFromHex(task.TaskID)
		if err != nil {
			return derp.Wrap(err, location, "Invalid taskID")
		}
	}

	// Set up filter and option arguments
	filter := bson.M{"_id": taskID}
	options := options.Update().SetUpsert(true)

	update := bson.M{"$set": task}

	// Update the database
	if _, err := storage.database.Collection(CollectionQueue).UpdateOne(timeout, filter, update, options); err != nil {
		return derp.Wrap(err, location, "Unable to save task to task queue")
	}

	// Silence is golden
	return nil
}

// DeleteTask removes a task from the queue
func (storage Storage) DeleteTask(taskID string) error {

	const location = "queue.deleteTask"
	timeout, cancel := timeoutContext(16)
	defer cancel()

	// Convert the taskID into a primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(taskID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid taskID")
	}

	// Remove the task from the task queue
	filter := bson.M{"_id": objectID}
	if _, err := storage.database.Collection(CollectionQueue).DeleteOne(timeout, filter); err != nil {
		return derp.Wrap(err, location, "Unable to delete task from task queue")
	}

	// Silence is acquiescence
	return nil
}

// LogFailure adds a task to the error log
func (storage Storage) LogFailure(task queue.Task) error {

	const location = "queue.logTask"
	timeout, cancel := timeoutContext(16)
	defer cancel()

	// Report the error (probably to the console)
	derp.Report(task.Error)

	// Add the task to the log
	if _, err := storage.database.Collection(CollectionLog).InsertOne(timeout, task); err != nil {
		return derp.Wrap(err, location, "Unable to add task to error log")
	}

	return nil
}

// GetTasks returns all tasks that are currently locked by this worker
func (storage Storage) GetTasks() ([]queue.Task, error) {

	const location = "mongo.Storage.queryTasks"

	// Create a timeout context for 16 seconds
	timeout, cancel := timeoutContext(16)
	defer cancel()

	result := make([]queue.Task, 0)
	lockID := primitive.NewObjectID()

	// Try to lock more tasks if we don't already have any
	if err := storage.lockTasks(timeout, lockID); err != nil {
		return result, derp.Wrap(err, location, "Unable to lock tasks")
	}

	// Find all tasks that are currently locked by this worker
	filter := bson.M{
		"lockId": lockID,
	}

	// Sort by startDate, and limit to the number of workers
	options := options.Find().SetSort(bson.M{"startDate": 1})

	// Query the database
	cursor, err := storage.database.Collection(CollectionQueue).Find(timeout, filter, options)

	if err != nil {
		return result, derp.Wrap(err, location, "Error finding tasks")
	}

	if err := cursor.All(timeout, &result); err != nil {
		return result, derp.Wrap(err, location, "Error decoding tasks")
	}

	return result, nil
}

// lockTasks assigns a set of tasks to the current worker
func (storage Storage) lockTasks(timeout context.Context, lockID primitive.ObjectID) error {

	const location = "mongo.Storage.lockTasks"

	// Identify the next set of tasks that COULD be run by this worker
	tasks, err := storage.pickTasks(timeout)

	if err != nil {
		return derp.Wrap(err, location, "Error picking tasks")
	}

	// Try to update these tasks IF they available to take
	filter := bson.M{
		"_id":         tasks,
		"timeoutDate": bson.M{"$lt": time.Now().Unix()},
	}

	// Assign to this worker and reset work counters
	update := bson.M{
		"$set": bson.M{
			"lockId":      lockID,
			"startDate":   time.Now().Unix(),
			"timeoutDate": time.Now().Add(time.Duration(storage.timeoutMinutes) * time.Minute).Unix(),
			"error":       nil,
		},
	}

	// Try to update all matching tasks.  We get what we get.
	if _, err := storage.database.Collection(CollectionQueue).UpdateMany(timeout, filter, update); err != nil {
		return derp.Wrap(err, location, "Error updating tasks")
	}

	return nil
}

// pickTasks identifies the next set of tasks that should be assigned to workers.
func (storage Storage) pickTasks(timeout context.Context) ([]primitive.ObjectID, error) {

	// Look for unassigned tasks, or tasks that have timed out
	filter := bson.M{
		"timeoutDate": bson.M{"$lt": time.Now().Unix()},
	}

	// Sort by startDate, and limit to the number of workers
	options := options.Find().
		SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "startDate", Value: 1}}).
		SetLimit(int64(storage.lockQuantity)).
		SetProjection(bson.M{
			"_id": 1,
		})

	// Query the database for matching Tasks
	cursor, err := storage.database.Collection(CollectionQueue).Find(timeout, filter, options)

	if err != nil {
		return nil, derp.Wrap(err, "mongo.Storage.lockTasks", "Error finding tasks")
	}

	// Decode the response into a slice
	temp := make([]struct {
		ID primitive.ObjectID `bson:"_id"`
	}, 0)

	if err := cursor.All(timeout, &temp); err != nil {
		return nil, derp.Wrap(err, "mongo.Storage.lockTasks", "Error decoding tasks")
	}

	// Extract the ObjectIDs from the slice
	result := make([]primitive.ObjectID, len(temp))
	for index, item := range temp {
		result[index] = item.ID
	}

	return result, nil
}
