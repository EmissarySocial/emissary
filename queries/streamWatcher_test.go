package queries

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestWatchStreams_SendsUpdatedAndChildUpdated pins the two messages a changed Stream produces
func TestWatchStreams_SendsUpdatedAndChildUpdated(t *testing.T) {

	database := newWatchTestDatabase(t)
	streamID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	messages := startMessageWatcher(t, database, WatchStreams)

	first := awaitMessage(t, messages, func() {
		upsertRecord(t, database, "Stream", streamID, bson.M{"parentId": parentID})
	})

	require.Equal(t, realtime.NewMessage_Updated(streamID), first)
	require.Equal(t, realtime.NewMessage_ChildUpdated(parentID), nextMessage(t, messages))
}

// TestWatchStreams_SkipsUnusableRecords pins that zero and undecodable Streams send nothing
func TestWatchStreams_SkipsUnusableRecords(t *testing.T) {

	database := newWatchTestDatabase(t)
	streamID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	messages := startMessageWatcher(t, database, WatchStreams)

	first := awaitMessage(t, messages, func() {

		// A zero StreamID, then a parentId that is not an ObjectID
		upsertRecord(t, database, "Stream", primitive.NilObjectID, bson.M{"parentId": parentID})
		upsertRecord(t, database, "Stream", primitive.NewObjectID(), bson.M{"parentId": "not-an-objectid"})

		// ...and finally, a Stream that works
		upsertRecord(t, database, "Stream", streamID, bson.M{"parentId": parentID})
	})

	require.Equal(t, realtime.NewMessage_Updated(streamID), first)
}

// TestSendMessage pins that a send either delivers or gives up when canceled, never hangs
func TestSendMessage(t *testing.T) {

	message := realtime.NewMessage_Updated(primitive.NewObjectID())

	t.Run("Delivers", func(t *testing.T) {
		result := make(chan realtime.Message, 1)

		sendMessage(context.Background(), result, message)

		require.Equal(t, message, <-result)
	})

	t.Run("GivesUpWhenCanceled", func(t *testing.T) {
		result := make(chan realtime.Message) // nobody ever reads this
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		requireReturns(t, func() { sendMessage(ctx, result, message) })
	})
}

/******************************************
 * Test Helpers
 ******************************************/

// messageWatcher is the signature shared by WatchStreams, WatchUsers, and WatchImports
type messageWatcher func(context.Context, data.Server, chan<- realtime.Message)

// startMessageWatcher runs a realtime watcher until the test ends, and returns its messages
func startMessageWatcher(t *testing.T, database *mongo.Database, watch messageWatcher) <-chan realtime.Message {

	t.Helper()

	messages := make(chan realtime.Message, 100)
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		watch(ctx, mongodb.NewServer(database), messages)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("watcher did not stop after its context was canceled")
		}
	})

	return messages
}

// awaitMessage calls write until the watcher sends a message, and returns that message.  Writing
// repeatedly covers the moments before the stream opens, when a single write would be missed.
func awaitMessage(t *testing.T, messages <-chan realtime.Message, write func()) realtime.Message {

	t.Helper()

	for deadline := time.Now().Add(watchTestTimeout); time.Now().Before(deadline); {

		write()

		select {
		case result := <-messages:
			return result
		case <-time.After(250 * time.Millisecond):
		}
	}

	t.Fatal("no message was sent")
	return realtime.Message{}
}

// nextMessage returns the next message sent
func nextMessage(t *testing.T, messages <-chan realtime.Message) realtime.Message {

	t.Helper()

	select {
	case result := <-messages:
		return result
	case <-time.After(watchTestTimeout):
		t.Fatal("no message was sent")
		return realtime.Message{}
	}
}

// upsertRecord replaces (or inserts) one document, which always produces an event with a document
func upsertRecord(t *testing.T, database *mongo.Database, collection string, recordID primitive.ObjectID, fields bson.M) {

	t.Helper()

	fields["_id"] = recordID
	fields["updated"] = time.Now().UnixNano()

	_, err := database.Collection(collection).ReplaceOne(context.Background(), bson.M{"_id": recordID}, fields, options.Replace().SetUpsert(true))
	require.Nil(t, err)
}

// insertRecordWithStringID inserts a document whose `_id` cannot be decoded into an ObjectID.
// The driver decodes a 24-character hex string as an ObjectID, so this one is neither.
func insertRecordWithStringID(t *testing.T, database *mongo.Database, collection string) {

	t.Helper()

	_, err := database.Collection(collection).InsertOne(context.Background(), bson.M{"_id": "not-an-objectid-" + primitive.NewObjectID().Hex()})
	require.Nil(t, err)
}
