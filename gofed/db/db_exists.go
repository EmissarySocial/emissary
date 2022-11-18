package db

import (
	"context"
	"net/url"

	"github.com/benpate/derp"
)

func (db *Database) Exists(_ context.Context, itemURL *url.URL) (exists bool, err error) {

	const location = "activitypub.Database.Exists"

	_, _, internalError := db.load(itemURL)

	if internalError == nil {
		return true, nil
	}

	if derp.NotFound(internalError) {
		return false, nil
	}

	return false, derp.Wrap(internalError, "Database.Exists", "Error checking if object exists", itemURL.String())
}
