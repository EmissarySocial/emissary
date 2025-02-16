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

	log.Trace().Msg("queries.WatchUsers")

	// Confirm that we're watching a mongo database
	m := mongoCollection(collection)

	if m == nil {
		return
	}

	log.Trace().Msg("queries.WatchUsers - Mongo Collection")

	// Get a change stream
	cs, err := m.Watch(ctx, mongo.Pipeline{})

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Watcher", "Unable to open Mongodb Change User"))
		return
	}

	log.Trace().Msg("queries.WatchUsers - Change Stream")

	// Send notifications whenever a User is changed
	for cs.Next(ctx) {

		log.Trace().Msg("queries.WatchUsers - Next")

		var event struct {
			User model.User `bson:"fullDocument"`
		}

		if err := cs.Decode(&event); err != nil {
			derp.Report(err)
			continue
		}

		log.Trace().Msg("queries.WatchUsers - Event")

		// Skip "zero" sreams
		if event.User.UserID.IsZero() {
			continue
		}

		result <- event.User.UserID
	}
}
