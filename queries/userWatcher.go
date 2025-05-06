package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// WatchUsers initiates a mongodb change stream to on every updates to User data objects
func WatchUsers(ctx context.Context, collection data.Collection, result chan<- primitive.ObjectID) {

	const location = "queries.WatchUsers"

	log.Trace().Msg(location)

	// Confirm that we're watching a mongo database
	m := mongoCollection(collection)

	if m == nil {
		return
	}

	log.Trace().Str("loc", location).Msg("Mongo Collection")

	// Get a change stream
	cs, err := m.Watch(ctx, mongo.Pipeline{})

	if err != nil {

		// MongoDB error 40573 indicates that we're running on a single node, not a replica set.
		if commandError, ok := err.(mongo.CommandError); ok {
			if commandError.Code == 40573 {
				return
			}
		}

		derp.Report(derp.Wrap(err, location, "Unable to open Mongodb Change User"))
		return
	}

	// Send notifications whenever a User is changed
	for cs.Next(ctx) {

		log.Trace().Str("loc", location).Msg("Next")

		var event struct {
			User model.User `bson:"fullDocument"`
		}

		if err := cs.Decode(&event); err != nil {
			derp.Report(err)
			continue
		}

		log.Trace().Str("loc", location).Msg("Event")

		// Skip "zero" sreams
		if event.User.UserID.IsZero() {
			continue
		}

		result <- event.User.UserID
	}
}
