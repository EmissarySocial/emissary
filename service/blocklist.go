package service

import (
	"github.com/benpate/activitystream/writer"
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionBlockList is the database collection where BlockLists are stored
const CollectionBlockList = "BlockList"

// BlockList manages all interactions with the BlockList collection
type BlockList struct {
	factory Factory
	session data.Session
}

// New creates a newly initialized BlockList that is ready to use
func (service BlockList) New() *model.BlockList {
	return &model.BlockList{
		BlockListID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the BlockLists who match the provided criteria
func (service BlockList) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.session.List(CollectionBlockList, criteria, options...)
}

// Load retrieves an BlockList from the database
func (service BlockList) Load(criteria expression.Expression) (*model.BlockList, *derp.Error) {

	actor := service.New()

	if err := service.session.Load(CollectionBlockList, criteria, actor); err != nil {
		return nil, derp.Wrap(err, "service.BlockList", "Error loading BlockList", criteria)
	}

	return actor, nil
}

// Save adds/updates an BlockList in the database
func (service BlockList) Save(actor *model.BlockList, note string) *derp.Error {

	if err := service.session.Save(CollectionBlockList, actor, note); err != nil {
		return derp.Wrap(err, "service.BlockList", "Error saving BlockList", actor, note)
	}

	return nil
}

// Delete removes an BlockList from the database (virtual delete)
func (service BlockList) Delete(actor *model.BlockList, note string) *derp.Error {

	if err := service.session.Delete(CollectionBlockList, actor, note); err != nil {
		return derp.Wrap(err, "service.BlockList", "Error deleting BlockList", actor, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service BlockList) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service BlockList) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service BlockList) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service BlockList) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.BlockList); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.BlockList", "Object is not a model.BlockList", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service BlockList) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.BlockList); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.BlockList", "Object is not a model.BlockList", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service BlockList) Close() {
	service.session.Close()
}

////////////////////////////////////////////////////
// CUSTOM FUNCTIONS
////////////////////////////////////////////////////

func (service BlockList) Block(id string, identity string, reason string, comment string) *derp.Error {

	blockListID, errr := primitive.ObjectIDFromHex(id)

	if errr != nil {
		return derp.New(404, "service.BlockList.Block", "Invalid BlockListID", id)
	}

	// TODO: Add mutex locks around this so that we avoid updating the same blocklist multiple times.

	blockList, err := service.Load(expression.New("_id", expression.OperatorEqual, blockListID))

	if err != nil {
		return derp.Wrap(err, "service.Blocklist.Block", "Can't Load Blocklist", blockListID)
	}

	if blockList.Add(identity, reason) {

		// Save the record to the database
		if err := service.Save(blockList, comment); err != nil {
			return err
		}

		// Create an ActivityStream event to publish
		event := writer.Block(
			writer.Person("", ""),
			writer.Object{},
		)

		// Try to publish the ActivityStream event to all listeners
		if err := service.factory.Publisher().Publish(event); err != nil {
			return derp.Wrap(err, "service.BlockList.Block", "Error publishing event.")
		}

		// TODO: publish the blocklist on GitHub (if designated public)
	}

	// TODO: complete mutex lock here.

	// Success!
	return nil
}
