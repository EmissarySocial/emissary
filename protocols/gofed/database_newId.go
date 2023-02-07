package gofed

import (
	"context"
	"net/url"

	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
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
func (db Database) NewID(ctx context.Context, t vocab.Type) (id *url.URL, err error) {

	spew.Dump("database.NewID")

	// If the object already has an ID, then just use that.
	if id := t.GetJSONLDId(); id != nil {
		return id.Get(), nil
	}

	// TODO: CRITICAL: otherwise, we need to make one based on the vocab type.
	switch t.GetTypeName() {
	case "Follow":

	}

	return nil, derp.NewInternalError("gofed.Database.NewID", "Unable to determine IRI for vocab.Type", t.GetTypeName())

	/*
		userID, err := getSignedInUserID(ctx)

		if err != nil {
			return nil, derp.Wrap(err, "gofed.Database.NewID", "Unable to determine UserID from context.Context")
		}

		spew.Dump(streams.Serialize(t))

		activityIRI := db.hostname + "/@" + userID.Hex() + "/out/" + primitive.NewObjectID().Hex()
		spew.Dump("activityIRI", activityIRI)
		return url.Parse(activityIRI)
	*/
}
