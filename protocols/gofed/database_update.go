package gofed

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
)

// Update is the same as Create except it is expected that the object already is in
// the database. The entity with the same id should be overwritten by the provided
// value. You do not need to worry about the ActivityPub specification talking about
// whether an Update means a partial-update or complete-replacement, as the library
// has already done this for you, so it is safe to simply replace the row.
func (db Database) Update(c context.Context, asType vocab.Type) error {

	// Convert the vocab.Type into a model.Activity
	updatedActivity, err := ToModel(asType, model.ActivityPlaceInbox)

	if err != nil {
		return derp.Wrap(err, "gofed.Database.Update", "Error converting to model object", asType)
	}

	// Determine the userID, location, and activityID from the URL
	activityURL := updatedActivity.URL()
	userID, place, activityID, err := ParsePath(activityURL)

	if err != nil {
		return derp.Wrap(err, "gofed.Database.Update", "Error parsing URL", updatedActivity)
	}

	// Try to load the existing activity
	existingActivity := model.NewActivity()
	if err := db.activityService.LoadByID(userID, place, activityID, &existingActivity); err != nil {
		return derp.Wrap(err, "gofed.Database.Update", "Error finding existing activity", updatedActivity)
	}

	// Update the existing activity with values from the caller
	existingActivity.UpdateWithActivity(&updatedActivity)

	// Save the activity back to the database.
	if err := db.activityService.Save(&existingActivity, "Updated by Go-Fed"); err != nil {
		return derp.Wrap(err, "gofed.Database.Update", "Error saving activity")
	}

	// Yussss.
	return nil
}
