package queries

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	mongodb "github.com/benpate/data-mongo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/******************************************
 * Domain Watcher Tests
 *
 * These pin how a Domain saved by one server reaches every
 * other server.  Each stores the record the way Emissary
 * does: one document whose `_id` is the zero ObjectID.
 ******************************************/

// TestWatchDomain_PublishesExistingRecordOnOpen pins the resync: the stored record is published
// as soon as the stream opens, before any change arrives.
func TestWatchDomain_PublishesExistingRecordOnOpen(t *testing.T) {

	database := newWatchTestDatabase(t)
	replaceDomain(t, database, bson.M{"label": "Existing"})

	published := startDomainWatcher(t, database)

	domain := awaitDomain(t, published)
	require.Equal(t, "Existing", domain.Label)
	require.True(t, domain.DomainID.IsZero())
}

// TestWatchDomain_PublishesReplacement pins the everyday case: a whole-document Save on another
// server reaches this one.
func TestWatchDomain_PublishesReplacement(t *testing.T) {

	database := newWatchTestDatabase(t)
	replaceDomain(t, database, bson.M{"label": "Before"})

	published := startDomainWatcher(t, database)
	require.Equal(t, "Before", awaitDomain(t, published).Label)

	replaceDomain(t, database, bson.M{"label": "After", "themeData": bson.M{"stylesheet": "body { color: red; }"}})

	domain := awaitDomain(t, published)
	require.Equal(t, "After", domain.Label)
	require.Equal(t, "body { color: red; }", domain.ThemeData.GetString("stylesheet"))
}

// TestWatchDomain_PublishesSetUpdate pins UpdateLookup: the upgrade runner writes the Domain with
// $set, and a plain update event carries no document at all.
func TestWatchDomain_PublishesSetUpdate(t *testing.T) {

	database := newWatchTestDatabase(t)
	replaceDomain(t, database, bson.M{"label": "Unchanged", "databaseVersion": 1})

	published := startDomainWatcher(t, database)
	require.Equal(t, uint(1), awaitDomain(t, published).DatabaseVersion)

	_, err := database.Collection("Domain").UpdateOne(context.Background(), bson.M{"_id": primitive.NilObjectID}, bson.M{"$set": bson.M{"databaseVersion": 35}})
	require.Nil(t, err)

	domain := awaitDomain(t, published)
	require.Equal(t, uint(35), domain.DatabaseVersion)
	require.Equal(t, "Unchanged", domain.Label)
}

// TestWatchDomain_NoRecordPublishesNothing pins that an empty collection publishes no blank Domain,
// which would replace a good cached record.
func TestWatchDomain_NoRecordPublishesNothing(t *testing.T) {

	database := newWatchTestDatabase(t)
	published := startDomainWatcher(t, database)

	// Write until something is published.  Whatever arrives, it must be the record written here.
	for deadline := time.Now().Add(watchTestTimeout); time.Now().Before(deadline); {

		replaceDomain(t, database, bson.M{"label": "Inserted"})

		select {
		case domain := <-published:
			require.Equal(t, "Inserted", domain.Label)
			return
		case <-time.After(250 * time.Millisecond):
		}
	}

	t.Fatal("no Domain was published")
}

// TestWatchDomain_SkipsUndecodableRecord pins that a record that cannot be decoded is reported,
// not published, and that the watcher keeps going.
func TestWatchDomain_SkipsUndecodableRecord(t *testing.T) {

	database := newWatchTestDatabase(t)
	replaceDomain(t, database, bson.M{"label": "Good"})

	published := startDomainWatcher(t, database)
	require.Equal(t, "Good", awaitDomain(t, published).Label)

	// A number cannot be decoded into the string Label
	replaceDomain(t, database, bson.M{"label": 12345})
	replaceDomain(t, database, bson.M{"label": "Recovered"})

	require.Equal(t, "Recovered", awaitDomain(t, published).Label)
}

// TestLoadDomain pins the read that resynchronizes the Domain on every open
func TestLoadDomain(t *testing.T) {

	t.Run("Publishes", func(t *testing.T) {
		database := newWatchTestDatabase(t)
		replaceDomain(t, database, bson.M{"label": "Loaded"})

		var result []model.WritableDomain
		err := loadDomain(context.Background(), database.Collection("Domain"), func(domain model.WritableDomain) { result = append(result, domain) })

		require.Nil(t, err)
		require.Len(t, result, 1)
		require.Equal(t, "Loaded", result[0].Label)
	})

	t.Run("InitializesMissingMaps", func(t *testing.T) {
		database := newWatchTestDatabase(t)
		replaceDomain(t, database, bson.M{"label": "Sparse"})

		var result model.WritableDomain
		err := loadDomain(context.Background(), database.Collection("Domain"), func(domain model.WritableDomain) { result = domain })

		require.Nil(t, err)
		require.NotNil(t, result.ThemeData)
		require.NotNil(t, result.Connections)
		require.NotNil(t, result.Data)
	})

	t.Run("NoRecord", func(t *testing.T) {
		database := newWatchTestDatabase(t)

		called := false
		err := loadDomain(context.Background(), database.Collection("Domain"), func(model.WritableDomain) { called = true })

		require.Nil(t, err)
		require.False(t, called)
	})

	t.Run("Undecodable", func(t *testing.T) {
		database := newWatchTestDatabase(t)
		replaceDomain(t, database, bson.M{"label": 12345})

		called := false
		err := loadDomain(context.Background(), database.Collection("Domain"), func(model.WritableDomain) { called = true })

		require.Error(t, err)
		require.False(t, called)
	})

	t.Run("Canceled", func(t *testing.T) {
		database := newWatchTestDatabase(t)
		replaceDomain(t, database, bson.M{"label": "Unreachable"})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		called := false
		err := loadDomain(ctx, database.Collection("Domain"), func(model.WritableDomain) { called = true })

		require.Error(t, err)
		require.False(t, called)
	})
}

/******************************************
 * Test Helpers
 ******************************************/

// startDomainWatcher runs WatchDomain until the test ends, and returns what it publishes
func startDomainWatcher(t *testing.T, database *mongo.Database) <-chan model.WritableDomain {

	t.Helper()

	published := make(chan model.WritableDomain, 100)
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		WatchDomain(ctx, mongodb.NewServer(database), func(domain model.WritableDomain) { published <- domain })
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("WatchDomain did not stop after its context was canceled")
		}
	})

	return published
}

// awaitDomain returns the next Domain published
func awaitDomain(t *testing.T, published <-chan model.WritableDomain) model.WritableDomain {

	t.Helper()

	select {
	case result := <-published:
		return result
	case <-time.After(watchTestTimeout):
		t.Fatal("no Domain was published")
		return model.WritableDomain{}
	}
}

// replaceDomain stores the Domain record the way Emissary does: one document with a zero `_id`
func replaceDomain(t *testing.T, database *mongo.Database, fields bson.M) {

	t.Helper()

	fields["_id"] = primitive.NilObjectID

	_, err := database.Collection("Domain").ReplaceOne(context.Background(), bson.M{"_id": primitive.NilObjectID}, fields, options.Replace().SetUpsert(true))
	require.Nil(t, err)
}
