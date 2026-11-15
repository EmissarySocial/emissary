package floodgatemongo

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/benpate/floodgate"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// testConnectString is the local replica set these tests expect.  RULE: a Go client reaching a
// single-node replica set from the host needs directConnection, or it hangs until timeout.
const testConnectString = "mongodb://127.0.0.1:27017/?directConnection=true&replicaSet=rs0"

const testHost = "remote.example"

// newTestDatabase creates a throwaway database with the Store's indexes, and skips the test
// when MongoDB is not running
func newTestDatabase(t testing.TB) *mongo.Database {

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

	database := client.Database("floodgatemongo_test_" + primitive.NewObjectID().Hex())
	require.NoError(t, SyncIndexes(ctx, database))

	// The test context is already cancelled by the time cleanup runs
	t.Cleanup(func() {
		_ = database.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	return database
}

// newTestStore returns a Store over a throwaway database
func newTestStore(t testing.TB) (Store, *mongo.Database) {

	t.Helper()

	database := newTestDatabase(t)
	return New(func() *mongo.Database { return database }), database
}

// TestStore_Load_UnknownHost verifies that a host with no record is a zero Host, not an error
func TestStore_Load_UnknownHost(t *testing.T) {

	store, _ := newTestStore(t)

	record, err := store.Load(t.Context(), testHost)
	require.NoError(t, err)
	require.Equal(t, floodgate.Host{}, record)
}

// TestStore_NoDatabase verifies that every method fails cleanly when the common database is not connected
func TestStore_NoDatabase(t *testing.T) {

	store := New(func() *mongo.Database { return nil })
	ctx := t.Context()
	now := time.Now()

	_, err := store.Load(ctx, testHost)
	require.Error(t, err)

	_, err = store.RangeBlocked(ctx)
	require.Error(t, err)

	_, err = store.RecordFailure(ctx, testHost, now, "2026-11-11")
	require.Error(t, err)

	require.Error(t, store.RecordSuccess(ctx, testHost, now))
	require.Error(t, store.SetPaused(ctx, testHost, true))
}

// TestStore_RecordFailure verifies that failures are counted, that a day is listed once, and
// that the record after each write is returned
func TestStore_RecordFailure(t *testing.T) {

	store, _ := newTestStore(t)
	ctx := t.Context()
	now := time.Unix(1_800_000_000, 0)

	first, err := store.RecordFailure(ctx, testHost, now, "2026-11-11")
	require.NoError(t, err)
	require.Equal(t, testHost, first.Host)
	require.Equal(t, 1, first.FailureCount)
	require.Equal(t, []string{"2026-11-11"}, first.FailureDates)
	require.Equal(t, now.Unix(), first.LastFailure)

	_, err = store.RecordFailure(ctx, testHost, now, "2026-11-11")
	require.NoError(t, err)

	third, err := store.RecordFailure(ctx, testHost, now, "2026-11-12")
	require.NoError(t, err)
	require.Equal(t, 3, third.FailureCount)
	require.Equal(t, []string{"2026-11-11", "2026-11-12"}, third.FailureDates)
}

// TestStore_RecordSuccess verifies that a success clears the trouble and keeps what the
// operator and the host set, and that it creates no record for an unknown host
func TestStore_RecordSuccess(t *testing.T) {

	store, database := newTestStore(t)
	ctx := t.Context()
	now := time.Unix(1_800_000_000, 0)

	require.NoError(t, store.RecordSuccess(ctx, "unknown.example", now))
	count, err := database.Collection(CollectionName).CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	require.Zero(t, count, "a healthy host must not cost a record")

	_, err = store.RecordFailure(ctx, testHost, now, "2026-11-11")
	require.NoError(t, err)
	require.NoError(t, store.MarkUnhealthy(ctx, testHost, now.Add(time.Minute), time.Minute))
	require.NoError(t, store.MarkDormant(ctx, testHost, now, now.Add(time.Hour)))
	require.NoError(t, store.SetSignatureFormat(ctx, testHost, "rfc9421"))
	require.NoError(t, store.SetNodeInfo(ctx, testHost, "mastodon", "4.7.0"))

	require.NoError(t, store.RecordSuccess(ctx, testHost, now.Add(time.Hour)))

	record, err := store.Load(ctx, testHost)
	require.NoError(t, err)
	require.False(t, record.IsUnhealthy)
	require.False(t, record.IsDormant())
	require.Zero(t, record.FailureCount)
	require.Empty(t, record.FailureDates)
	require.Zero(t, record.NextProbe)
	require.Equal(t, now.Add(time.Hour).Unix(), record.LastRevived)
	require.Equal(t, "rfc9421", record.SignatureFormat)
	require.Equal(t, "mastodon", record.Software)
}

// TestStore_MarkDormant_OnlyOnce verifies that a second mark does not restart the probe schedule
func TestStore_MarkDormant_OnlyOnce(t *testing.T) {

	store, _ := newTestStore(t)
	ctx := t.Context()
	now := time.Unix(1_800_000_000, 0)

	_, err := store.RecordFailure(ctx, testHost, now, "2026-11-11")
	require.NoError(t, err)

	require.NoError(t, store.MarkDormant(ctx, testHost, now, now.Add(time.Hour)))
	require.NoError(t, store.RecordProbeFailure(ctx, testHost, now.Add(2*time.Hour)))
	require.NoError(t, store.MarkDormant(ctx, testHost, now.Add(time.Minute), now.Add(9*time.Hour)))

	record, err := store.Load(ctx, testHost)
	require.NoError(t, err)
	require.Equal(t, now.Unix(), record.DormantSince)
	require.Equal(t, now.Add(2*time.Hour).Unix(), record.NextProbe)
	require.Equal(t, 1, record.ProbeCount)
}

// TestStore_SetRateLimit_KeepsTheLater verifies that a pause is created for an unknown host and is never shortened
func TestStore_SetRateLimit_KeepsTheLater(t *testing.T) {

	store, _ := newTestStore(t)
	ctx := t.Context()
	now := time.Unix(1_800_000_000, 0)

	require.NoError(t, store.SetRateLimit(ctx, testHost, now.Add(10*time.Minute)))
	require.NoError(t, store.SetRateLimit(ctx, testHost, now.Add(time.Minute)))

	record, err := store.Load(ctx, testHost)
	require.NoError(t, err)
	require.Equal(t, now.Add(10*time.Minute).Unix(), record.RateLimitUntil)

	require.NoError(t, store.SetRateLimit(ctx, testHost, now.Add(20*time.Minute)))

	record, err = store.Load(ctx, testHost)
	require.NoError(t, err)
	require.Equal(t, now.Add(20*time.Minute).Unix(), record.RateLimitUntil)
}

// hostNames returns the names of the provided hosts, sorted
func hostNames(t *testing.T, hosts func(func(floodgate.Host) bool), err error) []string {

	t.Helper()
	require.NoError(t, err)

	result := make([]string, 0)

	for host := range hosts {
		result = append(result, host.Host)
	}

	slices.Sort(result)
	return result
}

// TestStore_Ranges verifies which hosts are listed as blocked and which as due for a check
func TestStore_Ranges(t *testing.T) {

	store, _ := newTestStore(t)
	ctx := t.Context()
	now := time.Unix(1_800_000_000, 0)

	// A failing host, a rate-limited host and a healthy host are not blocked
	_, err := store.RecordFailure(ctx, "failing.example", now, "2026-11-11")
	require.NoError(t, err)
	require.NoError(t, store.SetRateLimit(ctx, "limited.example", now.Add(time.Hour)))

	require.NoError(t, store.SetPaused(ctx, "paused.example", true))
	require.NoError(t, store.SetPaused(ctx, "resumed.example", true))
	require.NoError(t, store.SetPaused(ctx, "resumed.example", false))

	for _, name := range []string{"due.example", "later.example"} {
		_, err := store.RecordFailure(ctx, name, now, "2026-11-11")
		require.NoError(t, err)
	}

	require.NoError(t, store.MarkDormant(ctx, "due.example", now, now.Add(time.Hour)))
	require.NoError(t, store.MarkDormant(ctx, "later.example", now, now.Add(3*time.Hour)))

	blocked, err := store.RangeBlocked(ctx)
	require.Equal(t, []string{"due.example", "later.example", "paused.example"}, hostNames(t, blocked, err))

	due, err := store.RangeProbesDue(ctx, now.Add(time.Hour))
	require.Equal(t, []string{"due.example"}, hostNames(t, due, err))
}

// TestStore_TwoServers verifies that two Stores sharing only the database count every failure
// that they report at the same moment
func TestStore_TwoServers(t *testing.T) {

	_, database := newTestStore(t)
	now := time.Unix(1_800_000_000, 0)

	servers := []Store{
		New(func() *mongo.Database { return database }),
		New(func() *mongo.Database { return database }),
	}

	var group sync.WaitGroup

	for index := range 100 {
		group.Go(func() {
			_, _ = servers[index%2].RecordFailure(context.Background(), testHost, now, "2026-11-11")
		})
	}

	group.Wait()

	record, err := servers[0].Load(t.Context(), testHost)
	require.NoError(t, err)
	require.Equal(t, 100, record.FailureCount)
	require.Equal(t, []string{"2026-11-11"}, record.FailureDates)
}

// TestStore_WithGate runs Floodgate's own rules over the MongoDB store: a host becomes unhealthy at the
// threshold, is tested after its cooldown, and recovers
func TestStore_WithGate(t *testing.T) {

	store, _ := newTestStore(t)
	ctx := t.Context()
	current := time.Unix(1_800_000_000, 0).UTC()
	gate := floodgate.New(store, floodgate.WithJitter(0), floodgate.WithClock(func() time.Time { return current }))

	for range 5 {
		require.NoError(t, gate.Failure(ctx, testHost))
	}

	decision, err := gate.Check(ctx, testHost)
	require.NoError(t, err)
	require.Equal(t, floodgate.ActionRetry, decision.Action)
	require.Equal(t, current.Add(time.Minute), decision.RetryAt)

	current = current.Add(time.Minute)

	decision, err = gate.Check(ctx, testHost)
	require.NoError(t, err)
	require.True(t, decision.CanSend(), "the cooldown has passed, so a trial goes through")

	require.NoError(t, gate.Success(ctx, decision.Host))

	record, err := store.Load(ctx, testHost)
	require.NoError(t, err)
	require.Equal(t, floodgate.StateHealthy, record.State(current))
	require.Zero(t, record.FailureCount)
}

// TestSyncIndexes verifies that both indexes exist, and that a second sync changes nothing
func TestSyncIndexes(t *testing.T) {

	_, database := newTestStore(t)
	ctx := t.Context()

	require.NoError(t, SyncIndexes(ctx, database))

	cursor, err := database.Collection(CollectionName).Indexes().List(ctx)
	require.NoError(t, err)

	indexes := make([]bson.M, 0)
	require.NoError(t, cursor.All(ctx, &indexes))

	names := make([]string, 0, len(indexes))

	for _, index := range indexes {
		name, _ := index["name"].(string)
		names = append(names, name)
	}

	require.ElementsMatch(t, []string{"_id_", "idx_Host_Dormant", "idx_Host_Paused"}, names)
}
