package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams/vocab"
)

// NewID allocates a new IRI for the Go-Fed library to use.  The library is in the
// process of creating a new ActivityStreams payload, and is calling this method to
// allocate a new IRI. You can inspect the context or the value, such as its type,
// in order to properly allocate an IRI meaningful to your application.
//
// Ensure that the newly allocated IRI can properly be fetched in another web handler
// by peers with proper authorization and authentication, which can be aided with
// pub.HandlerFunc.
func (db *Database) NewID(c context.Context, t vocab.Type) (id *url.URL, err error) {
	return nil, nil
}
