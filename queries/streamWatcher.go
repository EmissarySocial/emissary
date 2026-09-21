package queries

import (
	"context"

	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WatchStreams sends realtime updates whenever a Stream is inserted or replaced, until ctx is canceled
func WatchStreams(ctx context.Context, server data.Server, result chan<- realtime.Message) {

	const location = "queries.WatchStreams"

	watcher := changeWatcher{
		collection: "Stream",
		onDocument: func(ctx context.Context, document bson.Raw) {

			// Decode only the identifiers that the realtime messages need
			var stream struct {
				StreamID primitive.ObjectID `bson:"_id"`
				ParentID primitive.ObjectID `bson:"parentId"`
			}

			if err := bson.Unmarshal(document, &stream); err != nil {
				derp.Report(derp.Wrap(err, location, "Decoding Stream from change event"))
				return
			}

			// Skip "zero" streams
			if stream.StreamID.IsZero() {
				return
			}

			// Refresh pages showing this Stream, and pages listing its parent's children
			sendMessage(ctx, result, realtime.NewMessage_Updated(stream.StreamID))
			sendMessage(ctx, result, realtime.NewMessage_ChildUpdated(stream.ParentID))
		},
	}

	watcher.run(ctx, server)
}

// sendMessage delivers a realtime message, giving up if ctx is canceled first
func sendMessage(ctx context.Context, result chan<- realtime.Message, message realtime.Message) {

	select {
	case result <- message:
	case <-ctx.Done():
	}
}
