package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams/vocab"
)

// Followers returns a collection containing the provided Actor's followers.
// This must be the complete collection of followers for that Actor.
func (db Database) Followers(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	return nil, nil
}
