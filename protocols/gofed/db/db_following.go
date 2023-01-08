package db

import (
	"context"
	"net/url"

	"github.com/benpate/exp"
	"github.com/go-fed/activity/streams/vocab"
)

func (db *Database) Following(ctx context.Context, actorURL *url.URL) (vocab.ActivityStreamsCollection, error) {
	return db.getCollection("following", actorURL, exp.All())
}
