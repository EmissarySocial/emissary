package gofed

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// Delete removes the entity or row with the matching id.
func (db Database) Delete(c context.Context, id *url.URL) error {

	const location = "gofed.Database.Delete"

	// Validate the provided URL
	userID, container, activityStreamID, err := ParsePath(id)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing URL", id)
	}

	// Try to load the existing activity
	activity := model.NewActivityStream(model.ActivityStreamContainerUndefined)
	if err := db.activityStreamService.LoadFromContainer(userID, container, activityStreamID, &activity); err != nil {
		return derp.Wrap(err, location, "Error loading activity", id)
	}

	// Try to delete the existing activity
	if err := db.activityStreamService.Delete(&activity, "Deleted by go-fed"); err != nil {
		return derp.Wrap(err, location, "Error deleting activity", id)
	}

	return nil
}
