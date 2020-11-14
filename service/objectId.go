package service

import "go.mongodb.org/mongo-driver/bson/primitive"

// ZeroObjectID returns a primitive.ObjectID made of all zeroes.
func ZeroObjectID() primitive.ObjectID {

	result, _ := primitive.ObjectIDFromHex("000000000000000000000000")

	return result
}