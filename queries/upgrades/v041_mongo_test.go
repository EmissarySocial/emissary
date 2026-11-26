package upgrades

import (
	"context"
	"math"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// version41Fixture stores the Streams and outbox rows of one test, as an older server wrote them
type version41Fixture struct {
	t        *testing.T
	database *mongo.Database
	authorID primitive.ObjectID
}

// newVersion41Fixture returns a fixture on a throwaway database, with one author
func newVersion41Fixture(t *testing.T) version41Fixture {
	t.Helper()
	return version41Fixture{t: t, database: newUpgradeTestDatabase(t), authorID: primitive.NewObjectID()}
}

// stream stores a Stream by the fixture's author as an older server wrote it, with the
// provided properties over the defaults, and returns its ID
func (fixture version41Fixture) stream(templateID string, stateID string, fields bson.M) primitive.ObjectID {

	fixture.t.Helper()

	streamID := primitive.NewObjectID()

	record := bson.M{
		"_id":           streamID,
		"templateId":    templateID,
		"stateId":       stateID,
		"url":           "https://example.test/" + streamID.Hex(),
		"attributedTo":  bson.M{"userId": fixture.authorID},
		"publishDate":   int64(1_700_000_000),
		"unpublishDate": int64(math.MaxInt64),
		"deleteDate":    int64(0),
		"label":         "kept",
	}

	for key, value := range fields {
		record[key] = value
	}

	_, err := fixture.database.Collection("Stream").InsertOne(context.Background(), record)
	require.NoError(fixture.t, err)

	return streamID
}

// row stores one activity about a Stream in a User's outbox, in the named collection
func (fixture version41Fixture) row(collection string, actorID primitive.ObjectID, activityType string, streamID primitive.ObjectID, fields bson.M) {

	fixture.t.Helper()

	record := bson.M{
		"_id":          primitive.NewObjectID(),
		"actorType":    model.ActorTypeUser,
		"actorId":      actorID,
		"activityType": activityType,
		"objectId":     "https://example.test/" + streamID.Hex(),
	}

	for key, value := range fields {
		record[key] = value
	}

	_, err := fixture.database.Collection(collection).InsertOne(context.Background(), record)
	require.NoError(fixture.t, err)
}

// load reads one Stream back as a plain map
func (fixture version41Fixture) load(streamID primitive.ObjectID) bson.M {
	fixture.t.Helper()
	return loadRecord(fixture.t, fixture.database.Collection("Stream"), streamID)
}

// requireHeld fails unless the Stream is flagged and holds the placeholder fingerprint
func (fixture version41Fixture) requireHeld(streamID primitive.ObjectID, message string) {
	fixture.t.Helper()
	record := fixture.load(streamID)
	require.Equal(fixture.t, true, record["isOutbox"], message)
	require.Equal(fixture.t, model.StreamFingerprintPending, record["fingerprint"], message)
}

// requireUnsent fails unless the Stream stores a FALSE flag and no fingerprint
func (fixture version41Fixture) requireUnsent(streamID primitive.ObjectID, message string) {
	fixture.t.Helper()
	record := fixture.load(streamID)
	require.Equal(fixture.t, false, record["isOutbox"], message)
	require.NotContains(fixture.t, record, "fingerprint", message)
}

// TestVersion41_FlagsWhatFollowersHold verifies which outbox rows flag a Stream
func TestVersion41_FlagsWhatFollowersHold(t *testing.T) {

	fixture := newVersion41Fixture(t)

	created := fixture.stream("outbox-message", "published", nil)
	fixture.row("OutboxItem", fixture.authorID, "Create", created, nil)

	updated := fixture.stream("outbox-message", "default", nil)
	fixture.row("OutboxItem", fixture.authorID, "Update", updated, nil)

	legacy := fixture.stream("outbox-message", "published", nil)
	fixture.row("Outbox", fixture.authorID, "Create", legacy, nil)

	leftBehind := fixture.stream("outbox-message", "published", nil)
	fixture.row("Outbox2", fixture.authorID, "Create", leftBehind, nil)

	// A State without the flag keeps the flag that the row gives it
	article := fixture.stream("article", "published", nil)
	fixture.row("OutboxItem", fixture.authorID, "Create", article, nil)

	// A Stream in a flagged State that was never sent keeps its State, with the flag off
	neverSent := fixture.stream("outbox-message", "published", nil)

	// Unpublishing removed the Create and left a Delete under the same object
	unpublished := fixture.stream("outbox-message", "default", nil)
	fixture.row("OutboxItem", fixture.authorID, "Delete", unpublished, nil)

	// A row that was deleted is in no outbox
	deletedRow := fixture.stream("outbox-message", "published", nil)
	fixture.row("Outbox", fixture.authorID, "Create", deletedRow, bson.M{"deleteDate": int64(5)})

	// Someone else's outbox says nothing about this author's Stream
	someoneElse := fixture.stream("outbox-message", "published", nil)
	fixture.row("OutboxItem", primitive.NewObjectID(), "Create", someoneElse, nil)

	// A parent Stream's Announce is not the author's Create
	announced := fixture.stream("outbox-message", "published", nil)
	fixture.row("OutboxItem", fixture.authorID, "Create", announced, bson.M{"actorType": model.ActorTypeStream})

	// A deleted Stream gets neither the flag nor the placeholder
	deleted := fixture.stream("outbox-message", "published", bson.M{"deleteDate": int64(9)})
	fixture.row("OutboxItem", fixture.authorID, "Create", deleted, nil)

	// A fingerprint that the outbox task has already recorded stands
	recorded := fixture.stream("outbox-message", "published", bson.M{"isOutbox": true, "fingerprint": "abc123"})
	fixture.row("OutboxItem", fixture.authorID, "Create", recorded, nil)

	require.NoError(t, Version41(context.Background(), fixture.database))

	fixture.requireHeld(created, "a Create")
	fixture.requireHeld(updated, "an Update")
	fixture.requireHeld(legacy, "a row in the legacy Outbox")
	fixture.requireHeld(leftBehind, "a row left in Outbox2")
	fixture.requireHeld(article, "a State without the flag")

	fixture.requireUnsent(neverSent, "never sent")
	fixture.requireUnsent(unpublished, "a Delete alone")
	fixture.requireUnsent(deletedRow, "a deleted row")
	fixture.requireUnsent(someoneElse, "another User's row")
	fixture.requireUnsent(announced, "a Stream actor's row")

	require.Equal(t, "published", fixture.load(neverSent)["stateId"], "an unsent Stream keeps its State")
	require.Equal(t, "default", fixture.load(updated)["stateId"])

	deletedRecord := fixture.load(deleted)
	require.NotContains(t, deletedRecord, "fingerprint")
	require.NotEqual(t, true, deletedRecord["isOutbox"])

	require.Equal(t, "abc123", fixture.load(recorded)["fingerprint"])

	// Nothing else on a record is touched
	require.Equal(t, "kept", fixture.load(created)["label"])
}

// TestVersion41_MovesUnsentSongs verifies that a song leaves "published" only when it was
// never sent and its album is not on the website
func TestVersion41_MovesUnsentSongs(t *testing.T) {

	fixture := newVersion41Fixture(t)

	draftAlbum := fixture.stream("bandwagon-album", "default", nil)
	liveAlbum := fixture.stream("bandwagon-album", "published", nil)

	unsent := fixture.stream("bandwagon-song", "published", bson.M{"parentId": draftAlbum})
	orphan := fixture.stream("bandwagon-song", "published", bson.M{"parentId": primitive.NewObjectID()})

	sent := fixture.stream("bandwagon-song", "published", bson.M{"parentId": liveAlbum})
	fixture.row("OutboxItem", fixture.authorID, "Create", sent, nil)

	// A song with no row beneath an album that visitors can see stays visible
	rowLost := fixture.stream("bandwagon-song", "published", bson.M{"parentId": liveAlbum})

	// Another Template's "published" State is left alone
	message := fixture.stream("outbox-message", "published", bson.M{"parentId": draftAlbum})

	require.NoError(t, Version41(context.Background(), fixture.database))

	require.Equal(t, "default", fixture.load(unsent)["stateId"])
	require.Equal(t, "default", fixture.load(orphan)["stateId"])
	require.Equal(t, "published", fixture.load(sent)["stateId"])
	require.Equal(t, "published", fixture.load(rowLost)["stateId"])
	require.Equal(t, "published", fixture.load(message)["stateId"])

	fixture.requireUnsent(unsent, "a moved song")
	fixture.requireHeld(sent, "a sent song")
	fixture.requireUnsent(rowLost, "a song with no row")
}

// TestVersion41_ClearsTheDates verifies the publish dates of Streams and drafts
func TestVersion41_ClearsTheDates(t *testing.T) {

	fixture := newVersion41Fixture(t)

	never := fixture.stream("outbox-message", "default", bson.M{"publishDate": int64(math.MaxInt64)})
	dated := fixture.stream("outbox-message", "published", bson.M{"unpublishDate": int64(1_800_000_000)})
	deleted := fixture.stream("outbox-message", "default", bson.M{"publishDate": int64(math.MaxInt64), "deleteDate": int64(9)})

	drafts := fixture.database.Collection("StreamDraft")
	draftID := primitive.NewObjectID()
	_, err := drafts.InsertOne(context.Background(), bson.M{"_id": draftID, "publishDate": int64(math.MaxInt64), "unpublishDate": int64(math.MaxInt64)})
	require.NoError(t, err)

	require.NoError(t, Version41(context.Background(), fixture.database))

	require.Equal(t, int64(0), fixture.load(never)["publishDate"])
	require.NotContains(t, fixture.load(never), "unpublishDate")

	require.Equal(t, int64(1_700_000_000), fixture.load(dated)["publishDate"], "a real date is kept")
	require.NotContains(t, fixture.load(dated), "unpublishDate")

	require.Equal(t, int64(0), fixture.load(deleted)["publishDate"], "a deleted Stream's dates are cleared too")
	require.NotContains(t, fixture.load(deleted), "unpublishDate")

	draft := loadRecord(t, drafts, draftID)
	require.Equal(t, int64(0), draft["publishDate"])
	require.NotContains(t, draft, "unpublishDate")
	require.NotContains(t, draft, "isOutbox", "a draft gets no flag")
}

// TestVersion41_DropsWebhooks verifies that the Webhook collection is removed, and that a
// database without one is no error
func TestVersion41_DropsWebhooks(t *testing.T) {

	fixture := newVersion41Fixture(t)
	ctx := context.Background()

	_, err := fixture.database.Collection("Webhook").InsertOne(ctx, bson.M{"label": "gone"})
	require.NoError(t, err)

	require.NoError(t, Version41(ctx, fixture.database))

	names, err := fixture.database.ListCollectionNames(ctx, bson.M{})
	require.NoError(t, err)
	require.NotContains(t, names, "Webhook")

	require.NoError(t, Version41(ctx, fixture.database))
}

// TestVersion41_RunsTwice verifies that a second run changes nothing, including after the
// server has recorded real fingerprints and unpublished a Stream
func TestVersion41_RunsTwice(t *testing.T) {

	fixture := newVersion41Fixture(t)
	ctx := context.Background()
	streams := fixture.database.Collection("Stream")

	filled := fixture.stream("outbox-message", "published", nil)
	fixture.row("OutboxItem", fixture.authorID, "Create", filled, nil)

	draftAlbum := fixture.stream("bandwagon-album", "default", nil)
	song := fixture.stream("bandwagon-song", "published", bson.M{"parentId": draftAlbum})

	require.NoError(t, Version41(ctx, fixture.database))

	// The fill task replaces the placeholder
	_, err := streams.UpdateOne(ctx, bson.M{"_id": filled}, bson.M{"$set": bson.M{"fingerprint": "abc123"}})
	require.NoError(t, err)

	before := fixture.load(song)

	require.NoError(t, Version41(ctx, fixture.database))

	require.Equal(t, "abc123", fixture.load(filled)["fingerprint"], "a real fingerprint is never replaced")
	require.Equal(t, before, fixture.load(song))
}

// TestPlanVersion41 verifies that the plan counts by Template and State, and writes nothing
func TestPlanVersion41(t *testing.T) {

	fixture := newVersion41Fixture(t)

	sent := fixture.stream("outbox-message", "published", nil)
	fixture.row("OutboxItem", fixture.authorID, "Create", sent, nil)
	fixture.stream("outbox-message", "published", nil)
	fixture.stream("outbox-message", "default", bson.M{"publishDate": int64(math.MaxInt64)})
	fixture.stream("outbox-message", "default", bson.M{"unpublishDate": int64(1_600_000_000)})
	fixture.stream("outbox-message", "default", bson.M{"deleteDate": int64(9)})

	album := fixture.stream("bandwagon-album", "default", nil)
	fixture.stream("bandwagon-song", "published", bson.M{"parentId": album})

	plan, err := PlanVersion41(context.Background(), fixture.database)
	require.NoError(t, err)

	require.Equal(t, []Version41Count{
		{TemplateID: "bandwagon-album", StateID: "default", Total: 1},
		{TemplateID: "bandwagon-song", StateID: "published", Total: 1, Moved: 1},
		{TemplateID: "outbox-message", StateID: "default", Total: 2, Hidden: 2},
		{TemplateID: "outbox-message", StateID: "published", Total: 2, Held: 1, Flagged: 1},
	}, plan.Counts)

	// The plan is a dry run
	record := fixture.load(sent)
	require.NotContains(t, record, "isOutbox")
	require.NotContains(t, record, "fingerprint")
	require.Contains(t, record, "unpublishDate")
}

// TestVersion41Move_Applies verifies the rule that moves an unsent Stream out of a split State
func TestVersion41Move_Applies(t *testing.T) {

	move := version41Move{from: "published", to: "default", unlessParentFrom: "published"}

	require.True(t, move.applies("published", "default"))
	require.True(t, move.applies("published", ""), "a missing parent is not on the website")
	require.False(t, move.applies("published", "published"))
	require.False(t, move.applies("default", "default"), "a Stream that already moved stays")
}
