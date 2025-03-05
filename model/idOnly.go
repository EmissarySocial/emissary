package model

import (
	"github.com/benpate/rosetta/slice"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// IDOnly is a helper function for querying ONLY the ID of a batch of documents
type IDOnly struct {
	ID primitive.ObjectID `bson:"_id"`
}

func GetIDOnly(values []IDOnly) []primitive.ObjectID {
	return slice.Map(values, func(value IDOnly) primitive.ObjectID {
		return value.ID
	})
}
