package upgrades

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/derp/plugins"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestVersion39_CopiesTheLegacyOutbox runs the migration twice against a real MongoDB, over every
// shape of legacy row and every reason a row stays behind
func TestVersion39_CopiesTheLegacyOutbox(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()
	legacy := database.Collection("Outbox")
	target := database.Collection("OutboxItem")
	requireOutboxItemURLIndex(t, database)

	// insert stores one document in a collection
	insert := func(collection *mongo.Collection, document bson.M) primitive.ObjectID {
		t.Helper()
		if _, exists := document["_id"]; !exists {
			document["_id"] = primitive.NewObjectID()
		}
		_, err := collection.InsertOne(ctx, document)
		require.NoError(t, err)
		return document["_id"].(primitive.ObjectID)
	}

	const userURL = "https://example.com/@ben"
	const streamURL = "https://example.com/album"
	anonymous := bson.A{model.MagicGroupIDAnonymous}
	groupID := primitive.NewObjectID()

	userID := insert(database.Collection("User"), bson.M{"profileUrl": userURL})
	streamID := insert(database.Collection("Stream"), bson.M{"url": streamURL})

	// Rows that are copied
	publicCreate := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Create", "objectId": userURL + "/notes/1", "permissions": anonymous, "createDate": int64(1_500_000_000_000), "updateDate": int64(1_500_000_000_001)})
	restrictedUpdate := insert(legacy, bson.M{"actorId": streamID, "actorType": "Stream", "actorUrl": "", "activityType": "Update", "objectId": streamURL + "/song", "permissions": bson.A{groupID}, "createDate": int64(1_500_000_000_002)})
	like := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Like", "activityUrl": userURL + "/pub/liked/1", "objectId": "https://remote.example/notes/1", "permissions": anonymous})
	v070 := insert(legacy, bson.M{"parentId": userID, "parentType": "User", "url": userURL + "/notes/0", "createDate": int64(1_400_000_000_000)})
	sharedOlder := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Like", "activityUrl": userURL + "/pub/liked/2", "objectId": "https://remote.example/notes/2", "permissions": anonymous})

	// Rows that stay behind
	sharedNewer := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Like", "activityUrl": userURL + "/pub/liked/2", "objectId": "https://remote.example/notes/3", "permissions": anonymous})
	search := insert(legacy, bson.M{"actorId": primitive.NewObjectID(), "actorType": "Search", "actorUrl": "https://example.com/.search/1", "activityType": "Create", "objectId": "https://remote.example/notes/4", "permissions": anonymous})
	deleted := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Create", "objectId": userURL + "/notes/5", "permissions": anonymous, "deleteDate": int64(1_600_000_000_000)})
	orphan := insert(legacy, bson.M{"actorId": primitive.NewObjectID(), "actorType": "User", "actorUrl": "", "activityType": "Create", "objectId": "https://example.com/notes/6", "permissions": anonymous})
	undecodable := insert(legacy, bson.M{"actorId": "not an ObjectID", "actorType": "User", "actorUrl": userURL, "activityType": "Create", "permissions": anonymous})
	badPermissions := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Create", "permissions": bson.A{int32(7)}})

	// A row that OutboxItem already holds by _id, and one whose URL another OutboxItem holds
	present := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Create", "objectId": "legacy", "permissions": anonymous})
	insert(target, bson.M{"_id": present, "url": userURL + "/pub/outbox/" + present.Hex(), "objectId": "current"})

	collision := insert(legacy, bson.M{"actorId": userID, "actorType": "User", "actorUrl": userURL, "activityType": "Create", "objectId": userURL + "/notes/7", "permissions": anonymous})
	insert(target, bson.M{"url": userURL + "/pub/outbox/" + collision.Hex(), "objectId": "holder"})

	// Run twice: the second run must change nothing and report the same rows
	capture := captureUpgradeReports(t)
	require.NoError(t, Version39(ctx, database))
	require.Equal(t, []int{404, 409, 409, 500, 500}, capture.codes(), "the orphan, the shared URL, the taken URL, and the two unreadable rows are reported")

	first := allDocuments(t, target)
	require.Len(t, first, 7, "five copied, plus the two rows that were already there")

	require.NoError(t, Version39(ctx, database))
	require.Equal(t, first, allDocuments(t, target))

	legacyCount, err := legacy.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	require.Equal(t, int64(13), legacyCount, "the legacy rows stay where they are")

	// load reads one migrated row
	load := func(rowID primitive.ObjectID) model.OutboxItem {
		t.Helper()
		item := model.OutboxItem{}
		require.NoError(t, target.FindOne(ctx, bson.M{"_id": rowID}).Decode(&item))
		return item
	}

	// A public Create keeps its _id, dates, object and permissions, and names Public
	item := load(publicCreate)
	require.Equal(t, userID, item.ActorID)
	require.Equal(t, "User", item.ActorType)
	require.Equal(t, userURL+"/pub/outbox/"+publicCreate.Hex(), item.URL)
	require.Equal(t, "Create", item.ActivityType)
	require.Equal(t, userURL+"/notes/1", item.ObjectID)
	require.Equal(t, model.NewAnonymousPermissions(), item.Permissions)
	require.Equal(t, int64(1_500_000_000_000), item.CreateDate)
	require.Equal(t, int64(1_500_000_000_001), item.UpdateDate)
	require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, []string(item.Recipients))
	require.Equal(t, item.URL, item.Activity.GetString(vocab.PropertyID))
	require.Equal(t, userURL, item.Activity.GetString(vocab.PropertyActor))
	require.Equal(t, userURL+"/notes/1", item.Activity.GetString(vocab.PropertyObject))
	require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, item.Activity.GetSliceOfString(vocab.PropertyTo))

	// A Stream's row with no actorUrl takes the Stream's URL, and stays restricted
	item = load(restrictedUpdate)
	require.Equal(t, streamURL+"/pub/outbox/"+restrictedUpdate.Hex(), item.URL)
	require.Equal(t, streamURL, item.Activity.GetString(vocab.PropertyActor))
	require.Equal(t, model.Permissions{groupID}, item.Permissions)
	require.Empty(t, item.Recipients)
	require.False(t, item.IsVisibleTo(model.NewAnonymousPermissions(), ""))

	// A Like keeps its own activity URL
	require.Equal(t, userURL+"/pub/liked/1", load(like).URL)

	// BUG-304: a v0.7.0 row is read by its old names, and its actor's URL is looked up
	item = load(v070)
	require.Equal(t, userID, item.ActorID)
	require.Equal(t, "User", item.ActorType)
	require.Equal(t, "Create", item.ActivityType)
	require.Equal(t, userURL+"/notes/0", item.ObjectID)
	require.Equal(t, userURL+"/pub/outbox/"+v070.Hex(), item.URL)
	require.Equal(t, model.NewAnonymousPermissions(), item.Permissions)
	require.Equal(t, int64(1_400_000_000_000), item.CreateDate)

	// The older of two rows that share a URL keeps it
	require.Equal(t, "https://remote.example/notes/2", load(sharedOlder).ObjectID)

	// A row already in OutboxItem is never overwritten
	require.Equal(t, "current", load(present).ObjectID)

	// Rows that cannot be served are not copied
	for _, rowID := range []primitive.ObjectID{sharedNewer, search, deleted, orphan, undecodable, badPermissions, collision} {
		count, err := target.CountDocuments(ctx, bson.M{"_id": rowID})
		require.NoError(t, err)
		require.Zero(t, count, "row %s was copied", rowID.Hex())
	}
}

// TestVersion39_TwoServersAtOnce verifies that two servers running the slot together copy each row
// once, and neither fails
func TestVersion39_TwoServersAtOnce(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()
	requireOutboxItemURLIndex(t, database)

	rows := make([]any, 0, 200)

	for range 200 {
		rows = append(rows, bson.M{"_id": primitive.NewObjectID(), "actorId": primitive.NewObjectID(), "actorType": "User", "actorUrl": "https://example.com/@ben", "activityType": "Create", "objectId": "https://example.com/notes/1", "permissions": bson.A{model.MagicGroupIDAnonymous}})
	}

	_, err := database.Collection("Outbox").InsertMany(ctx, rows)
	require.NoError(t, err)

	capture := captureUpgradeReports(t)
	results := make(chan error, 2)
	var group sync.WaitGroup

	for range 2 {
		group.Go(func() { results <- Version39(ctx, database) })
	}

	group.Wait()
	close(results)

	for err := range results {
		require.NoError(t, err)
	}

	count, err := database.Collection("OutboxItem").CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	require.Equal(t, int64(200), count)
	require.Empty(t, capture.codes(), "a row the other server copied first is not a problem")
}

// TestVersion39_SharedURLWithoutTheIndex verifies that two rows sharing a URL leave only the older in
// OutboxItem on a database upgrading from v0.7.0, where idx_OutboxItem_URL is built after this slot
func TestVersion39_SharedURLWithoutTheIndex(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	older := primitive.NewObjectID()
	newer := primitive.NewObjectID()

	// row returns a Like that carries the shared activity URL
	row := func(rowID primitive.ObjectID) bson.M {
		return bson.M{"_id": rowID, "actorId": primitive.NewObjectID(), "actorType": "User", "actorUrl": "https://example.com/@ben", "activityType": "Like", "activityUrl": "https://example.com/@ben/pub/liked/1", "permissions": bson.A{model.MagicGroupIDAnonymous}}
	}

	// Stored newest first, so only the sort puts the older row ahead
	_, err := database.Collection("Outbox").InsertMany(ctx, []any{row(newer), row(older)})
	require.NoError(t, err)

	capture := captureUpgradeReports(t)
	require.NoError(t, Version39(ctx, database))
	require.Equal(t, []int{409}, capture.codes())

	documents := allDocuments(t, database.Collection("OutboxItem"))
	require.Len(t, documents, 1)
	require.Equal(t, older, documents[0]["_id"])

	// The index that sync.OutboxItem builds next can now be built
	requireOutboxItemURLIndex(t, database)
}

// TestLegacyOutboxCopier_Duplicate verifies how a duplicate key is settled: as present when the row's
// own _id is there, which is how a second server's copy looks, and otherwise as a URL held by another row
func TestLegacyOutboxCopier_Duplicate(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()
	copier := newLegacyOutboxCopier(database)
	capture := captureUpgradeReports(t)

	item := model.NewOutboxItem()
	item.URL = "https://example.com/@ben/pub/outbox/" + item.ActivityID.Hex()

	// No row holds the _id, so another row must hold the URL
	require.NoError(t, copier.duplicate(ctx, item))
	require.Equal(t, 1, copier.skipped)
	require.Equal(t, []int{409}, capture.codes())

	// Another server copied the row first
	_, err := database.Collection("OutboxItem").InsertOne(ctx, bson.M{"_id": item.ActivityID})
	require.NoError(t, err)
	require.NoError(t, copier.duplicate(ctx, item))
	require.Equal(t, 1, copier.present)
	require.Equal(t, []int{409}, capture.codes(), "a row that is present is not reported")
}

// TestLegacyOutboxCopier_DatabaseFailure verifies that a database failure while copying a row is
// returned, not skipped, so the slot fails and runs again on the next start
func TestLegacyOutboxCopier_DatabaseFailure(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	copier := newLegacyOutboxCopier(database)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// A row with no actorUrl must look its actor up, which fails
	row, err := bson.Marshal(bson.M{"_id": primitive.NewObjectID(), "actorId": primitive.NewObjectID(), "actorType": "User", "activityType": "Create"})
	require.NoError(t, err)
	require.Error(t, copier.copy(ctx, row))

	// The write fails, and so does settling a duplicate key
	item := model.NewOutboxItem()
	require.Error(t, copier.write(ctx, item))
	require.Error(t, copier.duplicate(ctx, item))

	require.Zero(t, copier.skipped, "a database failure is not a bad row")
}

// TestVersion39_NoLegacyCollection verifies that a Domain with no legacy outbox upgrades cleanly
func TestVersion39_NoLegacyCollection(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Version39(ctx, database))

	count, err := database.Collection("OutboxItem").CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	require.Zero(t, count)
}

// TestVersion39_DatabaseFailure verifies that a database that cannot be read fails the slot, so it
// runs again on the next start
func TestVersion39_DatabaseFailure(t *testing.T) {

	database := newUpgradeTestDatabase(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Error(t, Version39(ctx, database))
}

// requireOutboxItemURLIndex builds the unique URL index that sync.OutboxItem builds in production
func requireOutboxItemURLIndex(t *testing.T, database *mongo.Database) {

	t.Helper()

	index := mongo.IndexModel{
		Keys:    bson.D{{Key: "url", Value: 1}},
		Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"url": bson.M{"$gt": ""}}),
	}

	_, err := database.Collection("OutboxItem").Indexes().CreateOne(context.Background(), index)
	require.NoError(t, err)
}

// allDocuments returns every document in a collection, in _id order
func allDocuments(t *testing.T, collection *mongo.Collection) []bson.M {

	t.Helper()

	cursor, err := collection.Find(context.Background(), bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	require.NoError(t, err)

	documents := make([]bson.M, 0)
	require.NoError(t, cursor.All(context.Background(), &documents))

	return documents
}

// upgradeReportCapture is a derp Reporter that keeps every error reported while a test runs
type upgradeReportCapture struct {
	mutex  sync.Mutex
	errors []error
}

// Report keeps the error
func (capture *upgradeReportCapture) Report(err error) {
	capture.mutex.Lock()
	defer capture.mutex.Unlock()
	capture.errors = append(capture.errors, err)
}

// codes returns the derp code of every error reported so far, in ascending order
func (capture *upgradeReportCapture) codes() []int {

	capture.mutex.Lock()
	defer capture.mutex.Unlock()

	codes := make([]int, 0, len(capture.errors))

	for _, err := range capture.errors {
		codes = append(codes, derp.ErrorCode(err))
	}

	slices.Sort(codes)
	return codes
}

// captureUpgradeReports sends every derp.Report to a capture until the test ends, then restores
// derp's default reporter. The list is global, so a test that uses it must not run in parallel.
func captureUpgradeReports(t *testing.T) *upgradeReportCapture {

	t.Helper()

	capture := &upgradeReportCapture{}
	derp.Plugins.Set(capture)

	t.Cleanup(func() {
		derp.Plugins.Set(plugins.JSON{})
	})

	return capture
}
