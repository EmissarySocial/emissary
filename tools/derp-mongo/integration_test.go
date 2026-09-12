package derpmongo

import (
	"context"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// testConnectString is the local replica set these tests expect.  RULE: a Go client reaching a
// single-node replica set from the host needs directConnection, or it hangs until timeout.
const testConnectString = "mongodb://127.0.0.1:27017/?directConnection=true&replicaSet=rs0"

// newTestCollection creates a throwaway collection, and skips the test when MongoDB is not running
func newTestCollection(t *testing.T) *mongo.Collection {

	t.Helper()

	if testing.Short() {
		t.Skip("skipping a test that needs MongoDB")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(testConnectString))

	if err != nil {
		t.Skip("MongoDB is not available:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB is not responding:", err)
	}

	database := "derpmongo_test_" + primitive.NewObjectID().Hex()
	collection := client.Database(database).Collection("ErrorLog")

	t.Cleanup(func() {
		_ = client.Database(database).Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	return collection
}

func TestPluginReport_StampsTheStoredDocument(t *testing.T) {

	collection := newTestCollection(t)
	plugin := New(collection, mapof.Any{})

	inner := derp.NotFound("service.Inner.Load", "record not found")
	err := derp.Wrap(inner, "service.Outer.Do", "could not do the thing")

	plugin.Report(err)

	stored := bson.M{}
	require.NoError(t, collection.FindOne(t.Context(), bson.M{}).Decode(&stored))

	// Triage reads these two fields by name, so this is the contract between the packages
	assert.Equal(t, SignatureOf(err), stored["signature"])
	assert.Equal(t, string(StatusNew), stored["status"])

	assert.Equal(t, int32(404), stored["statusCode"])
	assert.Equal(t, "service.Inner.Load", stored["location"])
	assert.Equal(t, "record not found", stored["message"])

	assert.NotContains(t, stored, "statusNote", "a new record carries no decision")
	assert.NotContains(t, stored, "statusDate")
}

func TestPluginReport_OneSignaturePerDefect(t *testing.T) {

	collection := newTestCollection(t)
	plugin := New(collection, mapof.Any{})

	// The same defect against two different hosts is one item of work
	plugin.Report(derp.Internal("service.A", "lookup one.example.com: no such host"))
	plugin.Report(derp.Internal("service.A", "lookup two.example.net: no such host"))
	plugin.Report(derp.Internal("service.A", "something else entirely"))

	signatures, err := collection.Distinct(t.Context(), "signature", bson.M{})
	require.NoError(t, err)

	assert.Len(t, signatures, 2, "three records, two distinct defects")

	count, err := collection.CountDocuments(t.Context(), bson.M{"status": string(StatusNew)})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count, "every new record arrives undecided")
}

func TestPluginReport_HonorsExcludeList(t *testing.T) {

	collection := newTestCollection(t)
	plugin := New(collection, mapof.Any{"exclude-codes": []any{404}})

	plugin.Report(derp.NotFound("service.A", "not found"))
	plugin.Report(derp.Internal("service.A", "broken"))

	count, err := collection.CountDocuments(t.Context(), bson.M{})
	require.NoError(t, err)

	assert.Equal(t, int64(1), count, "the excluded status code never reached the database")
}
