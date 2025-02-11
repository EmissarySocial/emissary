package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// WatchStreams initiates a mongodb change stream to on every updates to Stream data objects
func WatchStreams(ctx context.Context, collection data.Collection, result chan<- primitive.ObjectID) {

	// Confirm that we're watching a mongo database
	m := mongoCollection(collection)

	if m == nil {
		return
	}

	// Get a change stream
	cs, err := m.Watch(ctx, mongo.Pipeline{})

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Watcher", "Unable to open Mongodb Change Stream"))
		return
	}

	// Send notifications whenever a Stream is changed
	for cs.Next(ctx) {

		var event struct {
			Stream model.Stream `bson:"fullDocument"`
		}

		if err := cs.Decode(&event); err != nil {
			derp.Report(err)
			continue
		}

		// Skip "zero" sreams
		if event.Stream.StreamID.IsZero() {
			continue
		}

		result <- event.Stream.StreamID
		result <- event.Stream.ParentID
	}
}
