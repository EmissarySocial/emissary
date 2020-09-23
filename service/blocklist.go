package service

import (
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
	factory    *Factory
	collection data.Collection
}

// New creates a newly initialized BlockList that is ready to use
func (service BlockList) New() *model.BlockList {
	return &model.BlockList{
		BlockListID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the BlockLists who match the provided criteria
func (service BlockList) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an BlockList from the database
func (service BlockList) Load(criteria expression.Expression) (*model.BlockList, error) {

	actor := service.New()

	if err := service.collection.Load(criteria, actor); err != nil {
		return nil, derp.Wrap(err, "service.BlockList", "Error loading BlockList", criteria)
	}

	return actor, nil
}

// Save adds/updates an BlockList in the database
func (service BlockList) Save(actor *model.BlockList, note string) error {

	if err := service.collection.Save(actor, note); err != nil {
		return derp.Wrap(err, "service.BlockList", "Error saving BlockList", actor, note)
	}

	return nil
}

// Delete removes an BlockList from the database (virtual delete)
func (service BlockList) Delete(actor *model.BlockList, note string) error {

	if err := service.collection.Delete(actor, note); err != nil {
		return derp.Wrap(err, "service.BlockList", "Error deleting BlockList", actor, note)
	}

	return nil
}

////////////////////////////////////////////////////
// CUSTOM FUNCTIONS
////////////////////////////////////////////////////

/*
// Block adds a particular ID/identity to the blocklist.
func (service BlockList) Block(id string, identity string, reason string, comment string) error {

	blockListID, errr := primitive.ObjectIDFromHex(id)

	if errr != nil {
		return derp.New(404, "service.BlockList.Block", "Invalid BlockListID", id)
	}

	// TODO: Add mutex locks around this so that we avoid updating the same blocklist multiple times.

	blockList, err := service.Load(expression.Equal("_id", blockListID))

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
*/
