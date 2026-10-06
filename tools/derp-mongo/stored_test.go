package derpmongo

import (
	"errors"
	"net/url"
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

func TestStoredError(t *testing.T) {

	for name, err := range unencodableErrors() {
		t.Run(name, func(t *testing.T) {

			require.False(t, isEncodable(err), "fixture must reproduce the failure")

			require.True(t, isEncodable(newRecord(err, 500)))
		})
	}
}

func TestStoredError_KeepsGoodParts(t *testing.T) {

	err := derp.Wrap(errors.New("boom"), "outer.Location", "outer message", "good detail", func() {})
	encodable, isEncodableError := StoredError(err).(encodableError)
	require.True(t, isEncodableError)

	require.Equal(t, "outer.Location", encodable.Location)
	require.Equal(t, "outer message", encodable.Message)
	require.Equal(t, "good detail", encodable.Details[0])
	require.Equal(t, "unencodable value of type func()", encodable.Details[1])
}

func TestStoredError_Nil(t *testing.T) {
	require.Nil(t, StoredError(nil))
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

// duplicateKeyError builds the chain data-mongo returns when an insert collides on a unique index
func duplicateKeyError(key string) error {

	writeException := mongo.WriteException{
		WriteErrors: mongo.WriteErrors{{
			Code:    11000,
			Message: "E11000 duplicate key error collection: test.User index: emailAddress_1 dup key: { emailAddress: \"" + key + "\" }",
		}},
	}

	inner := derp.Wrap(writeException, "data-mongo.Collection.Save", "Inserting object", derp.WithConflict())
	return derp.Wrap(inner, "service.User.Save", "Saving User")
}

func TestStoredError_CutsDupKey(t *testing.T) {

	record := newRecord(duplicateKeyError("person@example.com"), 409)

	stored, err := bson.Marshal(record)
	require.NoError(t, err)
	require.NotContains(t, string(stored), "person@example.com", "no stored form may quote the key")
	require.Contains(t, record.Message, "E11000 duplicate key error", "the rest of the message is kept")
	require.NotContains(t, record.Message, dupKeyMarker)
}

func TestStoredError_DupKeysShareOneSignature(t *testing.T) {

	first := newRecord(duplicateKeyError("first@example.com"), 409)
	second := newRecord(duplicateKeyError("second@example.com"), 409)

	require.Equal(t, first.Signature, second.Signature, "one defect, one item of work")
	require.Equal(t, SignatureOf(duplicateKeyError("first@example.com")), first.Signature)
}

func TestStoredError_KeepsEncodableForeignLayers(t *testing.T) {

	plain := errors.New("plain error")
	require.Equal(t, plain, StoredError(plain), "an encodable foreign error is stored as it is today")
}

func TestStoredError_LeavesOriginalAlone(t *testing.T) {

	err := duplicateKeyError("person@example.com")
	_ = StoredError(err)

	require.Contains(t, derp.RootMessage(err), "person@example.com", "the live error must not be modified")
}

func TestCutDupKey(t *testing.T) {
	require.Equal(t, "index: k_1", cutDupKey("index: k_1 dup key: { k: 1 }"))
	require.Equal(t, "no key here", cutDupKey("no key here"))
	require.Equal(t, "", cutDupKey("dup key: { k: 1 }"))
}

func TestStoredError_TypedNilDerpLayer(t *testing.T) {

	var inner *derp.Error
	err := derp.Wrap(inner, "outer.Location", "outer message")

	require.NotPanics(t, func() { newRecord(err, 500) })
	require.Nil(t, StoredError(inner))
}

func TestStoredError_TypedNilForeignLayer(t *testing.T) {

	var inner *url.Error
	err := derp.Wrap(inner, "outer.Location", "outer message")

	require.NotPanics(t, func() { newRecord(err, 500) })
	require.Nil(t, StoredError(inner))
}
