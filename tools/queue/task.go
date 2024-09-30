package queue

import (
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	TaskID     primitive.ObjectID `json:"id" bson:"_id"`
	QueueID    string             `json:"queueId" bson:"queueId"`
	Type       string             `json:"type" bson:"type"`
	Args       mapof.Any          `json:"arguments,omitempty" bson:"arguments,omitempty"`
	ActorID    primitive.ObjectID `json:"actorId" bson:"actorId"`
	WorkerID   primitive.ObjectID `json:"workerId" bson:"workerId"`
	JobType    string             `json:"jobType" bson:"jobType"`
	JobData    map[string]string  `json:"jobData" bson:"jobData"`
	Status     string             `json:"status" bson:"status"`
	CreateDate int64              `json:"createDate" bson:"createDate"`
	StartDate  int64              `json:"startDate" bson:"startDate"`
	EndDate    int64              `json:"endDate" bson:"endDate"`
}
