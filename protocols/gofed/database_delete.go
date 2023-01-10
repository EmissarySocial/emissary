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
	ownerID, _, _, err := ParseURL(id)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing URL", id)
	}

	// Try to load the existing activity
	activity := model.NewActivity()

	if err := db.activityService.LoadByURL(ownerID, id.String(), &activity); err != nil {
		return derp.Wrap(err, location, "Error loading activity", id)
	}

	// Try to delete the existing activity
	if err := db.activityService.Delete(&activity, "Deleted by go-fed"); err != nil {
		return derp.Wrap(err, location, "Error deleting activity", id)
	}

	return nil
}
