package db

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The library is in the process of creating a new ActivityStreams payload, and is calling
// this method to allocate a new IRI. You can inspect the context or the value, such as its type,
// in order to properly allocate an IRI meaningful to your application.
func (db *Database) NewID(ctx context.Context, item vocab.Type) (id *url.URL, err error) {

	itemType := item.GetTypeName()
	itemID := primitive.NewObjectID()

	urlString := "/.activitypub/" + itemType + "/" + itemID.Hex()
	return url.Parse(urlString)

	// Generate a new `id` for the ActivityStreams object `t`.

	// You can be fancy and put different types authored by different folks
	// along different paths. Or just generate a GUID. Implementation here
	// is left as an exercise for the reader.
}
