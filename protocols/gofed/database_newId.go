package gofed

import (
	"context"
	"net/url"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/streams/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NewID allocates a new IRI for the Go-Fed library to use.  The library is in the
// process of creating a new ActivityStreams payload, and is calling this method to
// allocate a new IRI. You can inspect the context or the value, such as its type,
// in order to properly allocate an IRI meaningful to your application.
//
// Ensure that the newly allocated IRI can properly be fetched in another web handler
// by peers with proper authorization and authentication, which can be aided with
// pub.HandlerFunc.
func (db Database) NewID(c context.Context, t vocab.Type) (id *url.URL, err error) {

	// TODO: CRITICAL: Do we need to distinguish inbox v. outbox
	// TODO: CRITICAL: How do we determine the UserID from the vocab.Type ??
	spew.Dump("database.NewID", t)

	activityIRI := db.hostname + "/@xxx/inbox/" + primitive.NewObjectID().Hex()
	return url.Parse(activityIRI)
}
