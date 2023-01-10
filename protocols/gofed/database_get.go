package gofed

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
)

// Get fetches the ActivityStreams object with id from the database. The streams.ToType
// function can turn any arbitrary JSON-LD literal into a vocab.Type for value.
func (db Database) Get(c context.Context, id *url.URL) (value vocab.Type, err error) {

	const location = "gofed.Database.Get"

	// Parse the URL
	ownerID, _, _, err := ParsePath(id)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing URL", id)
	}

	// Try to load the Activity from the database
	activity := model.NewActivity()

	if err := db.activityService.LoadByURL(ownerID, id.String(), &activity); err != nil {
		return nil, derp.Wrap(err, location, "Error loading activity", id)
	}

	// Encode result and return to caller
	return ToGoFed(&activity)
}
