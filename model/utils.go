package model

import "go.mongodb.org/mongo-driver/bson/primitive"

func objectID(value string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(value)
	return result
}
