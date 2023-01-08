package db

import (
	"context"
	"net/url"

	"github.com/benpate/derp"
)

func (db *Database) Delete(_ context.Context, itemURL *url.URL) error {

	const location = "service.activitypub.Database.Create"

	// Find the object in the database
	object, itemType, err := db.load(itemURL)

	if err != nil {
		return derp.Wrap(err, location, "Error loading object", itemURL)
	}

	// Get the corresponding ModelService to interact with the database
	modelService, err := db.factory.Model(itemType)

	if err != nil {
		return derp.Wrap(err, location, "No service found for item type", itemType)
	}

	// "Delete" the object from the database
	if err := modelService.ObjectDelete(object, "Delete via ActivityPub"); err != nil {
		return derp.Wrap(err, location, "Error deleting object", object)
	}

	// Success?!?
	return nil
}
