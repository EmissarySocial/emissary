package derpmongo

import (
	"errors"
	"syscall"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

// unencodableErrors are the two shapes production has failed to store
func unencodableErrors() map[string]error {

	// A WriteException with an empty Raw, as the driver returns on a duplicate key
	writeException := mongo.WriteException{
		WriteErrors: mongo.WriteErrors{{Code: 11000, Message: "E11000 duplicate key error"}},
	}

	// A ServerSelectionError wrapping a syscall.Errno, as an unreachable domain database returns
	serverSelection := topology.ServerSelectionError{Wrapped: syscall.ECONNREFUSED}

	method := func() string { return "id" }

	return map[string]error{
		"empty bson.Raw": derp.Wrap(writeException, "data-mongo.Collection.Save", "Inserting object"),
		"syscall.Errno":  derp.Wrap(serverSelection, "server.Factory.buildDomain", "Connecting to database"),
		"func detail":    derp.Wrap(errors.New("boom"), "server.Factory.refresh", "Refreshing domain", method),
	}
}

func TestMakeEncodable(t *testing.T) {

	for name, err := range unencodableErrors() {
		t.Run(name, func(t *testing.T) {

			require.False(t, isEncodable(newRecord(err, 500)), "fixture must reproduce the failure")

			record := newRecord(err, 500)
			record.Error = makeEncodable(record.Error)
			require.True(t, isEncodable(record))
		})
	}
}

func TestMakeEncodable_KeepsGoodParts(t *testing.T) {

	err := derp.Wrap(errors.New("boom"), "outer.Location", "outer message", "good detail", func() {})
	encodable := makeEncodable(err).(encodableError)

	require.Equal(t, "outer.Location", encodable.Location)
	require.Equal(t, "outer message", encodable.Message)
	require.Equal(t, "good detail", encodable.Details[0])
	require.Equal(t, "unencodable value of type func()", encodable.Details[1])
}

func TestMakeEncodable_Nil(t *testing.T) {
	require.Nil(t, makeEncodable(nil))
}

func TestPluginReport_StoresUnencodableErrors(t *testing.T) {

	collection := newTestCollection(t)
	plugin := New(collection, mapof.Any{})

	for _, err := range unencodableErrors() {
		plugin.Report(err)
	}

	count, err := collection.CountDocuments(t.Context(), bson.M{})
	require.NoError(t, err)
	require.Equal(t, int64(len(unencodableErrors())), count, "every unencodable error must still be stored")

	// Triage walks the chain by these names, so the rebuilt record must keep them
	stored := bson.M{}
	require.NoError(t, collection.FindOne(t.Context(), bson.M{"error.location": "data-mongo.Collection.Save"}).Decode(&stored))
	require.Contains(t, stored["error"], "wrappedvalue")
}
