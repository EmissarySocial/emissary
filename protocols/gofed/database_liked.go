package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// Liked returns the collection of Like records for a given actorIRI.
// This must be the complete collection of liked objects for that Actor.
func (db Database) Liked(c context.Context, actorIRI *url.URL) (liked vocab.ActivityStreamsCollection, err error) {

	// For now, we're not going to publish likes.  This may change some time in the future.
	result := streams.NewActivityStreamsCollection()
	return result, nil
}
