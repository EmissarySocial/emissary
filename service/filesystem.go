package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Filesystem manages all interactions with the Filesystem collection
type Filesystem struct {
	collection data.Collection
}

// NewFilesystem returns a fully populated Filesystem service
func NewFilesystem(collection data.Collection) Filesystem {
	return Filesystem{
		collection: collection,
	}
}

// New creates a newly initialized Filesystem that is ready to use
func (service Filesystem) New() model.Filesystem {
	return model.Filesystem{
		FilesystemID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Filesystems who match the provided criteria
func (service Filesystem) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Filesystem from the database
func (service Filesystem) Load(criteria exp.Expression, result *model.Filesystem) error {

	if err := service.collection.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Filesystem", "Error loading Filesystem", criteria)
	}

	return nil
}

// Save adds/updates an Filesystem in the database
func (service Filesystem) Save(attachment *model.Filesystem, note string) error {

	if err := service.collection.Save(attachment, note); err != nil {
		return derp.Wrap(err, "service.Filesystem", "Error saving Filesystem", attachment, note)
	}

	return nil
}

// Delete removes an Filesystem from the database (virtual delete)
func (service Filesystem) Delete(attachment *model.Filesystem, note string) error {

	if err := service.collection.Delete(attachment, note); err != nil {
		return derp.Wrap(err, "service.Filesystem", "Error deleting Filesystem", attachment, note)
	}

	return nil
}
