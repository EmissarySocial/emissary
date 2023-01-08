package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams/vocab"
)

// Get fetches the ActivityStreams object with id from the database. The streams.ToType
// function can turn any arbitrary JSON-LD literal into a vocab.Type for value.
func (db *Database) Get(c context.Context, id *url.URL) (value vocab.Type, err error) {
	return nil, nil
}
