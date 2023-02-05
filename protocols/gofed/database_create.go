package gofed

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
)

// Create stores the arbitrary ActivityStreams asType object into the database. It
// should be uniquely new to the database when examining its id property, and shouldn't
// overwrite any existing data.
//
// If needed, use streams.Serialize to turn the vocab.Type into a map[string]any.
func (db Database) Create(c context.Context, asType vocab.Type) error {

	const location = "gofed.Database.Create"

	// Convert the vocab.Type into a model.Activity
	activity, err := ToModel(asType, model.ActivityStreamContainerInbox)

	if err != nil {
		return derp.Wrap(err, location, "Error converting Type", asType)
	}

	// TODO: CRITICAL: What about other properties, like UserID???
	// TODO: CRITICAL: This will create duplicates / error out because the service isn't not searching for existing URLs.
	// Guessing this is only used for INBOUND activities..

	// Save the Activity to the database.
	if err := db.activityStreamService.Save(&activity, "Created by Go-Fed"); err != nil {
		return derp.Wrap(err, location, "Error saving activity", activity)
	}

	return nil
}
