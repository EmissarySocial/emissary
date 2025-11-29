package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/mongo"
)

// WatchImports initiates a mongodb change stream to on every updates to Import data objects
func WatchImports(ctx context.Context, server data.Server, result chan<- realtime.Message) {

	const location = "queries.WatchImports"

	// Connect to the database for as long as our refresh context is active
	session, err := server.Session(ctx)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to open database session"))
		return
	}

	// Confirm that we're watching a mongo database
	m := mongoCollection(session.Collection("Import"))

	if m == nil {
		return
	}

	// Get a change stream
	cs, err := m.Watch(ctx, mongo.Pipeline{})

	if err != nil {

		// MongoDB error 40573 indicates that we're running on a single node, not a replica set.
		if commandError, ok := err.(mongo.CommandError); ok {
			if commandError.Code == 40573 {
				return
			}
		}

		derp.Report(derp.Wrap(err, location, "Unable to open Mongodb Change Import"))
		return
	}

	// Send notifications whenever a Import is changed
	for cs.Next(ctx) {

		var event struct {
			Import model.Import `bson:"fullDocument"`
		}

		if err := cs.Decode(&event); err != nil {
			derp.Report(err)
			continue
		}

		// Skip "zero" sreams
		if event.Import.ImportID.IsZero() {
			continue
		}

		result <- realtime.NewMessage_ImportProgress(event.Import.ImportID)
	}
}
