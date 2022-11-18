package db

import (
	"context"
	"net/url"

	"github.com/benpate/exp"
	"github.com/go-fed/activity/streams/vocab"
)

func (db *Database) Followers(ctx context.Context, actorURL *url.URL) (vocab.ActivityStreamsCollection, error) {
	return db.getCollection("follwers", actorURL, exp.All())
}
