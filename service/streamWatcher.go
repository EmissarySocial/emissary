package service

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/mongo"
)

// WatchStreams initiates a mongodb change stream to on every updates to Stream data objects
func WatchStreams(collection *mongo.Collection, result chan<- model.Stream) {

	ctx := context.Background()

	cs, err := collection.Watch(ctx, mongo.Pipeline{})

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Watcher", "Unable to open Mongodb Change Stream"))
		return
	}

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

		result <- event.Stream
	}
}
