package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/gofed/common"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
)

func (db *Database) Get(_ context.Context, itemURL *url.URL) (value vocab.Type, err error) {

	const location = "activitypub.Database.Get"

	object, itemType, err := db.load(itemURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading object", itemURL)
	}

	// Convert the model object to an ActivityStream object
	return common.ToActivityStream(object, itemType)
}
