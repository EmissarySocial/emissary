package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/go-fed/activity/streams/vocab"
)

func (db *Database) Liked(ctx context.Context, actorURL *url.URL) (vocab.ActivityStreamsCollection, error) {
	return db.getCollection("reaction", actorURL, exp.Equal("type", model.ReactionTypeLike))
}
