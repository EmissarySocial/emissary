package upgrades

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests drive Version27 against a REAL MongoDB, because the parts it gets wrong are the parts
// a fake cannot check: whether a legacy `matchKey: null` actually backfills, whether duplicates
// clear, and whether the unique index can build afterwards. They skip when no database is reachable,
// so `go test ./...` still passes without one.

// directConnection bypasses replica-set discovery. The local demo's replica set advertises its
// member as "host.docker.internal", which does not resolve from outside the container, so a
// discovering client would hang on server selection instead of talking to the port in front of it.
const upgradeTestConnection = "mongodb://localhost:27017/?directConnection=true"

// newUpgradeTestDatabase connects to a local MongoDB and returns a throwaway database that is dropped
// when the test ends. The test is skipped when no database is reachable.
func newUpgradeTestDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(upgradeTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + upgradeTestConnection)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("Skipping: no MongoDB at " + upgradeTestConnection)
	}

	// A per-test database name keeps parallel runs (and the real demo data) well clear of this.
	database := client.Database("emissary_upgradetest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return database
}

// insertLegacyRule writes one raw Rule document with a NULL match key, exactly as a row created
// before the match-key engine landed would look. Returns the new RuleID.
func insertLegacyRule(t *testing.T, database *mongo.Database, userID primitive.ObjectID, ruleType string, trigger string, createDate int64) primitive.ObjectID {

	t.Helper()

	ruleID := primitive.NewObjectID()

	_, err := database.Collection("Rule").InsertOne(context.Background(), bson.M{
		"_id":        ruleID,
		"userId":     userID,
		"type":       ruleType,
		"trigger":    trigger,
		"matchKey":   nil,
		"action":     model.RuleActionBlock,
		"createDate": createDate,
		"deleteDate": 0,
	})

	require.Nil(t, err)
	return ruleID
}

// loadRuleMatchKey reads back the stored match key for one Rule.
func loadRuleMatchKey(t *testing.T, database *mongo.Database, ruleID primitive.ObjectID) string {

	t.Helper()

	result := struct {
		MatchKey string `bson:"matchKey"`
	}{}

	err := database.Collection("Rule").FindOne(context.Background(), bson.M{"_id": ruleID}).Decode(&result)

	require.Nil(t, err)
	return result.MatchKey
}

// countRules returns the number of Rules present in the collection.
func countRules(t *testing.T, database *mongo.Database) int64 {

	t.Helper()

	count, err := database.Collection("Rule").CountDocuments(context.Background(), bson.M{})

	require.Nil(t, err)
	return count
}

// buildMatchKeyIndex builds the unique {userId, matchKey} index -- the same one SyncDomainIndexes
// builds right after this upgrade runs. It is the acceptance test: the migration succeeds only if
// this build succeeds.
func buildMatchKeyIndex(ctx context.Context, database *mongo.Database) error {

	_, err := database.Collection("Rule").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "matchKey", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("idx_Rule_MatchKey"),
	})

	return err
}

// The exact reported reproduction: several admin rules (NilObjectID) all carrying `matchKey: null`
// with DISTINCT triggers. Before the fix they collided on (NilObjectID, null) and the unique index
// refused to build. After the migration each is backfilled to its own key, all survive, and the
// index builds.
func TestVersion27_ReportedReproduction(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	admin := primitive.NilObjectID

	first := insertLegacyRule(t, database, admin, model.RuleTypeDomain, "evil.com", 100)
	second := insertLegacyRule(t, database, admin, model.RuleTypeDomain, "spam.net", 200)
	third := insertLegacyRule(t, database, admin, model.RuleTypeActor, "https://bad.example/@troll", 300)

	require.Nil(t, Version27(ctx, database))

	// All three survive -- they were never duplicates, only un-keyed.
	require.Equal(t, int64(3), countRules(t, database))
	require.Equal(t, "DOMAIN:evil.com", loadRuleMatchKey(t, database, first))
	require.Equal(t, "DOMAIN:spam.net", loadRuleMatchKey(t, database, second))
	require.Equal(t, "ACTOR:https://bad.example/@troll", loadRuleMatchKey(t, database, third))

	// The unique index (built by SyncDomainIndexes right after this upgrade) must now build.
	require.Nil(t, buildMatchKeyIndex(ctx, database))
}

// Genuine duplicates -- one User, one trigger, two null rows -- collapse to the newest survivor, and
// the index builds.
func TestVersion27_DuplicatesCollapse(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()

	insertLegacyRule(t, database, userID, model.RuleTypeDomain, "evil.com", 100)
	newer := insertLegacyRule(t, database, userID, model.RuleTypeDomain, "evil.com", 200)

	require.Nil(t, Version27(ctx, database))

	require.Equal(t, int64(1), countRules(t, database))
	require.Equal(t, "DOMAIN:evil.com", loadRuleMatchKey(t, database, newer))
	require.Nil(t, buildMatchKeyIndex(ctx, database))
}

// A legacy "CONTENT" rule computes to no key: it can never match, so the migration deletes it and
// the index still builds on the now-empty collection.
func TestVersion27_InertRulesDeleted(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()

	insertLegacyRule(t, database, userID, "CONTENT", "some keyword", 100)
	insertLegacyRule(t, database, userID, "CONTENT", "another keyword", 200)

	require.Nil(t, Version27(ctx, database))

	require.Equal(t, int64(0), countRules(t, database))
	require.Nil(t, buildMatchKeyIndex(ctx, database))
}

// Rules owned by different Users are never confused with each other, even on an identical trigger.
func TestVersion27_LeavesOtherUsersAlone(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	userA := primitive.NewObjectID()
	userB := primitive.NewObjectID()

	ruleA := insertLegacyRule(t, database, userA, model.RuleTypeDomain, "evil.com", 100)
	ruleB := insertLegacyRule(t, database, userB, model.RuleTypeDomain, "evil.com", 100)

	require.Nil(t, Version27(ctx, database))

	require.Equal(t, int64(2), countRules(t, database))
	require.Equal(t, "DOMAIN:evil.com", loadRuleMatchKey(t, database, ruleA))
	require.Equal(t, "DOMAIN:evil.com", loadRuleMatchKey(t, database, ruleB))
	require.Nil(t, buildMatchKeyIndex(ctx, database))
}

// An empty Rule collection is a clean no-op: the migration succeeds and the index builds.
func TestVersion27_EmptyCollection(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	require.Nil(t, Version27(ctx, database))

	require.Equal(t, int64(0), countRules(t, database))
	require.Nil(t, buildMatchKeyIndex(ctx, database))
}
