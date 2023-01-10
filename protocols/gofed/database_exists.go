package gofed

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// Exists returns TRUE if the database has an entity or row with the id
func (db Database) Exists(c context.Context, id *url.URL) (exists bool, err error) {

	const location = "gofed.Database.Exists"

	// Validate the provided URL
	ownerID, activityType, activityID, err := ParsePath(id)

	if err != nil {
		return false, derp.Wrap(err, location, "Error parsing URL", id)
	}

	// Try to load the existing activity
	activity := model.NewActivity()
	err = db.activityService.LoadByID(ownerID, activityType, activityID, &activity)

	// No error means EXISTS
	if err == nil {
		return true, nil
	}

	// Not found error means NOT EXISTS
	if derp.NotFound(err) {
		return false, nil
	}

	// Otherwise, it's a legitimate error
	return false, derp.Wrap(err, location, "Error loading activity", id)
}
