package id

import (
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Converts a value into an ObjectID.  If the value cannot be converted, then a new ObjectID is returned
func ID(value interface{}) primitive.ObjectID {

	if id, ok := value.(primitive.ObjectID); ok {
		return id
	}

	stringValue := convert.String(value)

	if id, err := primitive.ObjectIDFromHex(stringValue); err == nil {
		return id
	}

	return primitive.NewObjectID()
}
