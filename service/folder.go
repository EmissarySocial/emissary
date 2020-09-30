package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionFolder is the database collection where Folders are stored
const CollectionFolder = "Folder"

// Folder service manages model.Folders.
type Folder struct {
	factory    *Factory
	collection data.Collection
}

// New returns a fully initialized Folder object
func (service Folder) New() *model.Folder {
	return &model.Folder{
		FolderID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Folders who match the provided criteria
func (service Folder) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Folder from the database
func (service Folder) Load(criteria expression.Expression) (*model.Folder, error) {

	stream := service.New()

	if err := service.collection.Load(criteria, stream); err != nil {
		return nil, derp.Wrap(err, "service.Folder", "Error loading Folder", criteria)
	}

	return stream, nil
}

// Save adds/updates an Folder in the database
func (service Folder) Save(stream *model.Folder, note string) error {

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, "service.Folder", "Error saving Folder", stream, note)
	}

	return nil
}

// Delete removes an Folder from the database (virtual delete)
func (service Folder) Delete(stream *model.Folder, note string) error {

	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "service.Folder", "Error deleting Folder", stream, note)
	}

	return nil
}

/////////////////////////////////////////
// Custom Queries

// LoadByToken retrieves a single Folder from the database, using the token as a key
func (service Folder) LoadByToken(token string) (*model.Folder, error) {
	return service.Load(expression.Equal("token", token))
}

// ListByParent retrieves all Folders that match the provided ParentID
func (service Folder) ListByParent(parentID primitive.ObjectID) (data.Iterator, error) {
	return service.List(expression.Equal("parentId", parentID))
}
