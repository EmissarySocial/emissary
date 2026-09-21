package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WatchImports sends realtime updates whenever an Import is inserted or replaced, until ctx is canceled
func WatchImports(ctx context.Context, server data.Server, result chan<- realtime.Message) {

	const location = "queries.WatchImports"

	watcher := changeWatcher{
		collection: "Import",
		onDocument: func(ctx context.Context, document bson.Raw) {

			// Decode only the identifier that the realtime message needs
			var record struct {
				ImportID primitive.ObjectID `bson:"_id"`
			}

			if err := bson.Unmarshal(document, &record); err != nil {
				derp.Report(derp.Wrap(err, location, "Decoding Import from change event"))
				return
			}

			// Skip "zero" imports
			if record.ImportID.IsZero() {
				return
			}

			// Refresh pages showing this Import's progress
			sendMessage(ctx, result, realtime.NewMessage_ImportProgress(record.ImportID))
		},
	}

	watcher.run(ctx, server)
}
