// Package id provides some nifty tools for manipulating mongodb objectIDs.
// They are not baked into rosetta (where they belong, conceptually) because
// it would introduce a dependency on the mongodb driver, which does not
// make sense for rosetta.  So, they're here.  Deal with it :)
package id

import (
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Converts a value into an ObjectID.  If the value cannot be converted, then a new ObjectID is returned
func ID(value any) primitive.ObjectID {

	if id, ok := value.(primitive.ObjectID); ok {
		return id
	}

	stringValue := convert.String(value)

	if id, err := primitive.ObjectIDFromHex(stringValue); err == nil {
		return id
	}

	return primitive.NewObjectID()
}
