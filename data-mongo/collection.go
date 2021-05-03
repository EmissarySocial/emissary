package mongodb

import (
	"context"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Collection wraps a mongodb.Collection with all of the methods required by the data.Collection interface
type Collection struct {
	collection *mongo.Collection
	context    context.Context
}

// List retrieves a group of objects from the database
func (c Collection) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {

	criteriaBSON := ExpressionToBSON(criteria)

	// TODO: translate options into mongodb options...

	cursor, err := c.collection.Find(c.context, criteriaBSON)

	if err != nil {
		return NewIterator(c.context, cursor), derp.New(derp.CodeInternalError, "mongodb.List", "Error Listing Objects", err.Error(), criteria, criteriaBSON, options)
	}

	iterator := NewIterator(c.context, cursor)

	return iterator, nil
}

// Load retrieves a single object from the database
func (c Collection) Load(criteria exp.Expression, target data.Object) error {

	criteriaBSON := ExpressionToBSON(criteria)

	if err := c.collection.FindOne(c.context, criteriaBSON).Decode(target); err != nil {

		var errorCode int

		if err.Error() == "mongo: no documents in result" {
			errorCode = derp.CodeNotFoundError
		} else {
			errorCode = derp.CodeInternalError
		}

		return derp.New(errorCode, "mongodb.Load", "Error loading object", err.Error(), criteria, criteriaBSON, target)
	}

	return nil
}

// Save inserts/updates a single object in the database.
func (c Collection) Save(object data.Object, note string) error {

	object.SetUpdated(note)

	// If new, then INSERT the object
	if object.IsNew() {
		object.SetCreated(note)

		if _, err := c.collection.InsertOne(c.context, object); err != nil {
			return derp.New(derp.CodeInternalError, "mongodb.Save", "Error inserting object", err.Error(), object)
		}

		return nil
	}

	// Fall through to here means UPDATE object

	objectID, err := primitive.ObjectIDFromHex(object.ID())

	if err != nil {
		return derp.New(derp.CodeInternalError, "mongodb.Save", "Error generating objectID", err, object)
	}

	filter := bson.M{
		"_id":                objectID,
		"journal.deleteDate": 0,
	}

	update := bson.M{"$set": object}

	if _, err := c.collection.UpdateOne(c.context, filter, update); err != nil {
		return derp.New(derp.CodeInternalError, "mongodb.Save", "Error saving object", err.Error(), filter, update)
	}

	return nil
}

// Delete removes a single object from the database, using a "virtual delete"
func (c Collection) Delete(object data.Object, note string) error {

	if object.IsNew() {
		return derp.New(derp.CodeBadRequestError, "mongo.Delete", "Cannot delete a new object", object, note)
	}

	// Use virtual delete to mark this object as deleted.
	object.SetDeleted(note)
	return c.Save(object, note)
}

// HardDelete physically removes an object from the database.
func (c Collection) HardDelete(object data.Object) error {

	filter := bson.M{"_id": object.ID()}

	_, err := c.collection.DeleteOne(c.context, filter)

	return derp.Wrap(err, "mondodb.HardDelete", "Error performing hard delete", object)
}

// Mongo returns the underlying mongodb collection for libraries that need to bypass this abstraction.
func (c Collection) Mongo() *mongo.Collection {
	return c.collection
}
