package upgrades

import (
	"context"
	"sync"
	"testing"

	derpconsole "github.com/EmissarySocial/emissary/tools/derp-console"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// upgradeReports is a derp.Reporter that keeps every error reported to it
type upgradeReports struct {
	lock   sync.Mutex
	errors []error
}

// Report records one error
func (reports *upgradeReports) Report(err error) {
	reports.lock.Lock()
	defer reports.lock.Unlock()
	reports.errors = append(reports.errors, err)
}

// TestForEachRecord_FailedSaveOmitsTheRecord requires that a record which cannot be written back
// is reported by collection and ID, never by its contents
func TestForEachRecord_FailedSaveOmitsTheRecord(t *testing.T) {

	// BUG-173: this error attached the whole record, and the upgrades walk Domain records,
	// which hold the domain's private key.
	database := newUpgradeTestDatabase(t)
	collection := database.Collection("Domain")
	ctx := context.Background()

	reports := &upgradeReports{}
	derp.SetPlugins(reports)
	t.Cleanup(func() { derp.SetPlugins(derpconsole.New()) })

	// A unique index makes the write-back fail for real, on the second record
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.M{"hostname": 1}, Options: options.Index().SetUnique(true)})
	require.NoError(t, err)

	_, err = collection.InsertMany(ctx, []any{
		bson.M{"hostname": "one.example.com", "privateKey": "fake-key-one"},
		bson.M{"hostname": "two.example.com", "privateKey": "fake-key-two"},
	})
	require.NoError(t, err)

	// Rename every record to one hostname, so every write after the first collides
	err = ForEachRecord(collection, func(record mapof.Any) bool {
		record["hostname"] = "one.example.com"
		return true
	})
	require.NoError(t, err)

	require.Len(t, reports.errors, 1)
	require.Equal(t, "Saving record", derp.Message(reports.errors[0]))
	// The driver's write error cannot be encoded as BSON, so each layer is checked on its own
	secretcheck.RequireAbsentFromEachLayer(t, reports.errors[0], "fake-key-one")
	secretcheck.RequireAbsentFromEachLayer(t, reports.errors[0], "fake-key-two")
}
