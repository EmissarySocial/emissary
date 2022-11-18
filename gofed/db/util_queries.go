package db

import (
	"net/url"

	"github.com/EmissarySocial/emissary/gofed/common"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/go-fed/activity/streams/vocab"
)

/***********************************
 * Helper Methods
 ***********************************/

func (db *Database) load(id *url.URL) (data.Object, string, error) {

	const location = "activitypub.Database.load"

	userID, itemType, itemID, err := parseURL(id)

	if err != nil {
		return nil, itemType, derp.Wrap(err, location, "Error parsing URL", id)
	}

	// Get the service for this kind of item
	modelService, err := db.factory.Model(itemType)

	if err != nil {
		return nil, itemType, derp.Wrap(err, location, "No service found for item type", itemType)
	}

	// Try to load the record from the database
	object, err := modelService.ObjectLoad(exp.Equal("_id", itemID).AndEqual("userId", userID))

	if err != nil {
		return nil, itemType, derp.Wrap(err, location, "Error loading object", id)
	}

	return object, itemType, nil
}

func (db *Database) save(item vocab.Type, comment string) error {

	const location = "service.activitypub.Database.save"

	// Extract important values from the item ID
	_, itemType, _, err := common.ParseItem(item)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing Item", item)
	}

	// Get the service for this kind of item
	modelService, err := db.factory.Model(itemType)

	if err != nil {
		return derp.Wrap(err, location, "No service found for item type", itemType)
	}

	// Convert the ActivityStream object to a model object
	object, err := ToModelObject(item)

	if err != nil {
		return derp.Wrap(err, location, "Error converting item to model object", item)
	}

	// Save the object to the database
	if err := modelService.ObjectSave(object, comment); err != nil {
		return derp.Wrap(err, location, "Error saving object", object)
	}

	// Success?!?
	return nil
}

func (db *Database) queryAllURLs(modelType string, actorURL *url.URL, criteria exp.Expression) ([]string, error) {

	const location = "service.activitypub.queryAllIRIsByURL"

	type hasAllURLs interface {
		QueryAllURLS(exp.Expression) ([]string, error)
	}

	// Parse the URL into a UserID
	userID, _, _, err := common.ParseURL(actorURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing URL", actorURL)
	}

	// Build the query criteria
	criteria = criteria.
		AndEqual("userId", userID).
		AndEqual("journal.deleteDate", 0)

	// Get the corresponding Model Service
	modelService, err := db.factory.Model(modelType)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error getting model service", modelType)
	}

	// Confirm that the model service has a "QueryAllURLs" method
	if queryable, ok := modelService.(hasAllURLs); ok {
		return queryable.QueryAllURLS(criteria)
	}

	return nil, derp.NewInternalError(location, "ModelService does not implement QueryAllURLS() method")
}
