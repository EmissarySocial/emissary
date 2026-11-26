package upgrades

import (
	"slices"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// decodeLegacyRow stores a document as BSON and reads it back as a legacy row, as Version39 does
func decodeLegacyRow(t *testing.T, document bson.M) legacyOutboxRow {

	t.Helper()

	encoded, err := bson.Marshal(document)
	require.NoError(t, err)

	row := legacyOutboxRow{}
	require.NoError(t, bson.Unmarshal(encoded, &row))

	return row
}

// TestLegacyOutboxRow_Message verifies that every shape of legacy row is read as a current message
func TestLegacyOutboxRow_Message(t *testing.T) {

	rowID := primitive.NewObjectID()
	actorID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()

	t.Run("a current row keeps every value", func(t *testing.T) {

		row := decodeLegacyRow(t, bson.M{
			"_id":          rowID,
			"actorId":      actorID,
			"actorType":    model.ActorTypeUser,
			"actorUrl":     "https://example.com/@ben",
			"activityType": vocab.ActivityTypeLike,
			"activityUrl":  "https://example.com/@ben/pub/liked/1",
			"objectId":     "https://remote.example/notes/1",
			"permissions":  bson.A{groupID},
			"createDate":   int64(1_500_000_000_000),
			"updateDate":   int64(1_600_000_000_000),

			// The v0.7.0 names lose to the current ones
			"parentId":   primitive.NewObjectID(),
			"parentType": model.ActorTypeStream,
			"url":        "https://other.example/1",
		})

		message, err := row.message()
		require.NoError(t, err)

		require.Equal(t, model.OutboxMessage{
			OutboxMessageID: rowID,
			ActorID:         actorID,
			ActorType:       model.ActorTypeUser,
			ActorURL:        "https://example.com/@ben",
			ActivityType:    vocab.ActivityTypeLike,
			ActivityURL:     "https://example.com/@ben/pub/liked/1",
			ObjectID:        "https://remote.example/notes/1",
			Permissions:     model.Permissions{groupID},
			Journal:         journal.Journal{CreateDate: 1_500_000_000_000, UpdateDate: 1_600_000_000_000},
		}, message)
	})

	t.Run("a v0.7.0 row reads its old names, and is a public Create", func(t *testing.T) {

		// BUG-304: v0.7.0 named the actor as the parent, the object as `url`, and had no permissions
		row := decodeLegacyRow(t, bson.M{
			"_id":        rowID,
			"parentId":   actorID,
			"parentType": model.ActorTypeUser,
			"url":        "https://example.com/notes/1",
			"createDate": int64(1_400_000_000_000),
		})

		message, err := row.message()
		require.NoError(t, err)
		require.Equal(t, actorID, message.ActorID)
		require.Equal(t, model.ActorTypeUser, message.ActorType)
		require.Equal(t, "https://example.com/notes/1", message.ObjectID)
		require.Equal(t, vocab.ActivityTypeCreate, message.ActivityType)
		require.Equal(t, model.NewAnonymousPermissions(), message.Permissions)
		require.Empty(t, message.ActorURL, "the copier looks the actor's URL up")
		require.Equal(t, int64(1_400_000_000_000), message.CreateDate)
	})

	t.Run("a row with a type but no activityType is a Create", func(t *testing.T) {
		message, err := decodeLegacyRow(t, bson.M{"_id": rowID, "type": "Note", "permissions": bson.A{}}).message()
		require.NoError(t, err)
		require.Equal(t, vocab.ActivityTypeCreate, message.ActivityType)
	})

	// test checks the permissions that one stored value reads as
	test := func(name string, permissions any, expected model.Permissions) {
		t.Run(name, func(t *testing.T) {
			message, err := decodeLegacyRow(t, bson.M{"_id": rowID, "permissions": permissions}).message()
			require.NoError(t, err)
			require.Equal(t, expected, message.Permissions)
		})
	}

	test("an empty list is served to no one", bson.A{}, model.Permissions{})
	test("a null list is served to no one", nil, model.Permissions{})
	test("a list that is not a list is served to no one", "Public", model.Permissions{})
	test("anonymous stays anonymous", bson.A{model.MagicGroupIDAnonymous}, model.NewAnonymousPermissions())

	t.Run("a list of something other than IDs is refused", func(t *testing.T) {
		message, err := decodeLegacyRow(t, bson.M{"_id": rowID, "permissions": bson.A{int32(7)}}).message()
		require.Error(t, err)
		require.Equal(t, model.OutboxMessage{}, message)
	})
}

// TestOutboxItemFromLegacy verifies that a migrated row serves what the legacy outbox served
func TestOutboxItemFromLegacy(t *testing.T) {

	rowID := primitive.NewObjectID()
	actorID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()

	message := model.OutboxMessage{
		OutboxMessageID: rowID,
		ActorID:         actorID,
		ActorType:       model.ActorTypeUser,
		ActorURL:        "https://example.com/@ben",
		ActivityType:    vocab.ActivityTypeCreate,
		ObjectID:        "https://example.com/notes/1",
		Permissions:     model.NewAnonymousPermissions(),
		Journal:         journal.Journal{CreateDate: 1_500_000_000_000, UpdateDate: 1_500_000_000_001},
	}

	t.Run("a public row names Public", func(t *testing.T) {

		item := outboxItemFromLegacy(message)
		url := "https://example.com/@ben/pub/outbox/" + rowID.Hex()

		require.Equal(t, rowID, item.ActivityID)
		require.Equal(t, actorID, item.ActorID)
		require.Equal(t, model.ActorTypeUser, item.ActorType)
		require.Equal(t, url, item.URL)
		require.Equal(t, vocab.ActivityTypeCreate, item.ActivityType)
		require.Equal(t, "https://example.com/notes/1", item.ObjectID)
		require.Equal(t, model.NewAnonymousPermissions(), item.Permissions)
		require.Equal(t, journal.Journal{CreateDate: 1_500_000_000_000, UpdateDate: 1_500_000_000_001}, item.Journal)
		require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, []string(item.Recipients))

		// The object stays a URL, as the legacy outbox served it (OUTBOX2-OMNIBUS §1.21)
		require.Equal(t, url, item.Activity.GetString(vocab.PropertyID))
		require.Equal(t, "https://example.com/@ben", item.Activity.GetString(vocab.PropertyActor))
		require.Equal(t, vocab.ActivityTypeCreate, item.Activity.GetString(vocab.PropertyType))
		require.Equal(t, "https://example.com/notes/1", item.Activity.GetString(vocab.PropertyObject))
		require.Equal(t, "2017-07-14T02:40:00Z", item.Activity.GetString(vocab.PropertyPublished))
		require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, item.Activity.GetSliceOfString(vocab.PropertyTo))

		require.True(t, item.IsVisibleTo(model.NewAnonymousPermissions(), ""), "anyone reads a public row")
	})

	t.Run("a restricted row names no one", func(t *testing.T) {

		restricted := message
		restricted.Permissions = model.Permissions{groupID}
		item := outboxItemFromLegacy(restricted)

		require.Empty(t, item.Recipients)
		require.Empty(t, item.Activity.GetSliceOfString(vocab.PropertyTo))
		require.False(t, item.IsVisibleTo(model.NewAnonymousPermissions(), ""), "a stranger reads nothing")
		require.True(t, item.IsVisibleTo(model.Permissions{groupID}, ""), "a member of the group reads it")
		require.True(t, item.IsVisibleTo(model.NewPermissions(), "https://example.com/@ben"), "its actor reads it")
	})

	t.Run("a row with its own activity URL keeps it", func(t *testing.T) {

		liked := message
		liked.ActivityType = vocab.ActivityTypeLike
		liked.ActivityURL = "https://example.com/@ben/pub/liked/1"
		item := outboxItemFromLegacy(liked)

		require.Equal(t, "https://example.com/@ben/pub/liked/1", item.URL)
		require.Equal(t, "https://example.com/@ben/pub/liked/1", item.Activity.GetString(vocab.PropertyID))
	})
}

// TestInsertFields verifies that a stored row carries every field but its _id, which the upsert's
// filter supplies
func TestInsertFields(t *testing.T) {

	item := model.NewOutboxItem()
	item.URL = "https://example.com/@ben/pub/outbox/1"

	fields, err := insertFields(item)
	require.NoError(t, err)

	keys := make([]string, 0, len(fields))

	for _, element := range fields {
		keys = append(keys, element.Key)
	}

	require.NotContains(t, keys, "_id")
	require.Subset(t, keys, []string{"actorId", "actorType", "url", "activityType", "objectId", "activity", "recipients", "permissions", "createDate", "updateDate", "deleteDate"})
}

// FuzzLegacyOutboxRow verifies that any stored row is read without panicking, that a current name
// always wins over its v0.7.0 name, and that a row it reads becomes an OutboxItem with the row's own
// _id, which names Public exactly when anyone may read it and can always be stored
func FuzzLegacyOutboxRow(f *testing.F) {

	// Fields are fuzzed rather than bytes, because the slot reads only documents that MongoDB
	// returns, and the driver's own Unmarshal and Validate panic on some malformed bytes.
	f.Add("User", "https://example.com/@ben", "Create", "", "https://example.com/notes/1", "", "", uint8(3), int64(1_500_000_000_000), "https://example.com/@ben")
	f.Add("", "", "", "", "", "Stream", "https://example.com/notes/1", uint8(0), int64(1_400_000_000_000), "https://example.com/album")
	f.Add("User", "", "Like", "https://example.com/@ben/pub/liked/1", "https://remote.example/notes/1", "", "", uint8(4), int64(0), "")
	f.Add("Search", "https://example.com/.search/1", "Create", "", "", "User", "x", uint8(5), int64(-1), "\xff")
	f.Add("", "", "", "", "", "", "", uint8(1), int64(0), "")
	f.Add("", "", "", "", "", "", "", uint8(6), int64(0), "")

	f.Fuzz(func(t *testing.T, actorType string, actorURL string, activityType string, activityURL string, objectID string, parentType string, url string, permissionsKind uint8, createDate int64, lookedUpURL string) {

		rowID := primitive.NewObjectID()
		parentID := primitive.NewObjectID()

		document := bson.M{
			"_id":          rowID,
			"actorType":    actorType,
			"actorUrl":     actorURL,
			"activityType": activityType,
			"activityUrl":  activityURL,
			"objectId":     objectID,
			"parentId":     parentID,
			"parentType":   parentType,
			"url":          url,
			"createDate":   createDate,
		}

		// Every way a row has stored its permissions, and two ways it never should have
		switch permissionsKind % 7 {
		case 1:
			document["permissions"] = nil
		case 2:
			document["permissions"] = bson.A{}
		case 3:
			document["permissions"] = bson.A{model.MagicGroupIDAnonymous}
		case 4:
			document["permissions"] = bson.A{primitive.NewObjectID()}
		case 5:
			document["permissions"] = bson.A{int32(7)}
		case 6:
			document["permissions"] = "Public"
		}

		encoded, err := bson.Marshal(document)
		require.NoError(t, err)

		row := legacyOutboxRow{}

		if err := bson.Unmarshal(encoded, &row); err != nil {
			return
		}

		// RULE: A row is read, or refused with nothing read
		message, err := row.message()

		if err != nil {
			require.Equal(t, model.OutboxMessage{}, message)
			return
		}

		require.Equal(t, rowID, message.OutboxMessageID)
		require.Equal(t, parentID, message.ActorID, "a row with no actorId takes its parentId")
		require.Equal(t, first(actorType, parentType), message.ActorType)
		require.Equal(t, first(objectID, url), message.ObjectID)
		require.Equal(t, first(activityType, vocab.ActivityTypeCreate), message.ActivityType)
		require.NotNil(t, message.Permissions)
		require.Equal(t, permissionsKind%7 == 0 || permissionsKind%7 == 3, message.Permissions.IsAnonymous(), "only a missing or anonymous list is public")

		// Every row that is read becomes an item that names itself, and that can be stored
		message.ActorURL = first(actorURL, lookedUpURL)
		item := outboxItemFromLegacy(message)

		require.Equal(t, rowID, item.ActivityID)
		require.Equal(t, item.URL, item.Activity.GetString(vocab.PropertyID))
		require.Equal(t, message.Permissions.IsAnonymous(), slices.Contains(item.Recipients, vocab.NamespaceActivityStreamsPublic))

		fields, err := insertFields(item)
		require.NoError(t, err)
		require.NotEmpty(t, fields)
	})
}

// first returns the first value that is not empty
func first(values ...string) string {

	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
