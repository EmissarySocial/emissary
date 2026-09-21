package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WatchUsers sends realtime updates whenever a User is inserted or replaced, until ctx is canceled
func WatchUsers(ctx context.Context, server data.Server, result chan<- realtime.Message) {

	const location = "queries.WatchUsers"

	watcher := changeWatcher{
		collection: "User",
		onDocument: func(ctx context.Context, document bson.Raw) {

			// Decode only the identifier that the realtime message needs
			var user struct {
				UserID primitive.ObjectID `bson:"_id"`
			}

			if err := bson.Unmarshal(document, &user); err != nil {
				derp.Report(derp.Wrap(err, location, "Decoding User from change event"))
				return
			}

			// Skip "zero" users
			if user.UserID.IsZero() {
				return
			}

			// Refresh pages showing this User
			sendMessage(ctx, result, realtime.NewMessage_Updated(user.UserID))
		},
	}

	watcher.run(ctx, server)
}
