package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams/vocab"
)

// Following returns a collection containing the provided Actor's following.
// This must be the complete collection of following for that Actor.
func (db *Database) Following(c context.Context, actorIRI *url.URL) (following vocab.ActivityStreamsCollection, err error) {
	return nil, nil
}
