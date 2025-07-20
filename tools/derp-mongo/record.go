package derpmongo

import "go.mongodb.org/mongo-driver/bson/primitive"

type Record struct {
	RecordID   primitive.ObjectID `bson:"_id"`
	StatusCode int                `bson:"statusCode"`
	Location   string             `bson:"location"`
	Message    string             `bson:"message"`
	Error      error              `bson:"error"`
	CreateDate primitive.DateTime `bson:"createDate"`
}
