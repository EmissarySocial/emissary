package service

import (
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests run the export and import against a real MongoDB, because an import must keep a
// create date that the database driver would otherwise stamp. They skip when no database is reachable.

// importedObjects is an importedObjectFinder that knows a fixed set of objects, by remote URL
type importedObjects map[string]importedObject

// importedObject is one object that an import brought over
type importedObject struct {
	userID      primitive.ObjectID
	jsonLD      mapof.Any
	publishDate int64
}

// findImportedObject returns the object for the remote URL, when it belongs to the User
func (objects importedObjects) findImportedObject(_ data.Session, userID primitive.ObjectID, remoteURL string) (mapof.Any, int64, bool) {

	object, found := objects[remoteURL]

	if !found || object.userID != userID {
		return nil, 0, false
	}

	return object.jsonLD, object.publishDate, true
}

// outbox2ImportTest holds an Outbox2 on new.example and the User who is importing into it
type outbox2ImportTest struct {
	outbox2FollowUpTest
	user    model.User
	objects importedObjects
}

// newOutbox2ImportTest returns an Outbox2 service that imports for one User
func newOutbox2ImportTest(t *testing.T) outbox2ImportTest {

	t.Helper()

	test := newOutbox2FollowUpTest(t)
	objects := importedObjects{}
	test.service.importedObjects = objects

	user := model.NewUser()
	user.UserID = test.userID

	return outbox2ImportTest{outbox2FollowUpTest: test, user: user, objects: objects}
}

// exported returns the JSON that the old server exported for an OutboxItem
func exported(t *testing.T, item model.OutboxItem) []byte {

	t.Helper()

	document, err := json.Marshal(item)
	require.NoError(t, err)

	return document
}

// remoteItem returns a public Create as the old server stored it, in the outbox of the old User
func remoteItem(objectURL string) model.OutboxItem {

	const oldActor = "https://old.example/@111111111111111111111111"

	item := model.NewOutboxItem()
	item.ActorType = model.ActorTypeUser
	item.ActorID = primitive.NewObjectID()
	item.URL = oldActor + "/pub/outbox/" + item.ActivityID.Hex()
	item.ActivityType = "Forged"
	item.ObjectID = "https://forged.example/object"
	item.Permissions = model.NewAnonymousPermissions()
	item.CreateDate = 1_600_000_000_000
	item.UpdateDate = 1_600_000_000_999
	item.DeleteDate = 1_600_000_001_000
	item.Activity = mapof.Any{
		vocab.PropertyID:     item.URL,
		vocab.PropertyType:   vocab.ActivityTypeCreate,
		vocab.PropertyActor:  oldActor,
		vocab.PropertyTo:     []string{vocab.NamespacePublic},
		vocab.PropertyCC:     []string{oldActor + "/pub/followers", "https://remote.example/@friend"},
		vocab.PropertyBCC:    []string{oldActor},
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: objectURL, vocab.PropertyType: vocab.ObjectTypeNote},
	}

	return item
}

// TestOutbox2_Export_RoundTrip verifies that an export lists a User's activities oldest first, and
// that a document carries the journal and the blind recipients
func TestOutbox2_Export_RoundTrip(t *testing.T) {

	test := newOutbox2ImportTest(t)
	query := outbox2QueryTest{service: test.service, session: test.session}
	otherID := primitive.NewObjectID()
	anonymous := model.NewAnonymousPermissions()

	withBlind := func(item *model.OutboxItem) {
		item.Activity = mapof.Any{vocab.PropertyBCC: []string{"https://remote.example/@blind"}}
	}

	newer := query.save(t, model.ActorTypeUser, test.userID, 200, anonymous, nil)
	older := query.save(t, model.ActorTypeUser, test.userID, 100, anonymous, withBlind)
	query.save(t, model.ActorTypeUser, otherID, 150, anonymous, nil)
	query.save(t, model.ActorTypeStream, test.userID, 160, anonymous, nil)

	ids, err := test.service.ExportCollection(test.session, test.userID)
	require.NoError(t, err)
	require.Len(t, ids, 2)
	require.Equal(t, older.ActivityID, ids[0].ID)
	require.Equal(t, newer.ActivityID, ids[1].ID)

	document, err := test.service.ExportDocument(test.session, test.userID, older.ActivityID)
	require.NoError(t, err)

	decoded := model.NewOutboxItem()
	require.NoError(t, json.Unmarshal([]byte(document), &decoded))
	require.Equal(t, int64(100), decoded.CreateDate, "the export carries the create date")
	require.Equal(t, []string{"https://remote.example/@blind"}, decoded.Activity.GetSliceOfString(vocab.PropertyBCC))

	// One User cannot export another's activity by naming its id
	_, err = test.service.ExportDocument(test.session, otherID, older.ActivityID)
	require.Error(t, err)
}

// TestOutbox2_Import verifies that an imported activity belongs to the importing User under this
// server's URLs, keeps its date, and is never delivered
func TestOutbox2_Import(t *testing.T) {

	test := newOutbox2ImportTest(t)
	remote := remoteItem("https://old.example/notes/1")
	importItem := model.NewImportItem()

	require.NoError(t, test.service.Import(test.session, nil, &importItem, &test.user, exported(t, remote)))

	// The import record maps the old identity to the new one
	require.Equal(t, remote.ActivityID, importItem.RemoteID)
	require.Equal(t, remote.URL, importItem.RemoteURL)
	require.NotEqual(t, remote.ActivityID, importItem.LocalID)

	item := model.NewOutboxItem()
	require.NoError(t, test.service.LoadByID(test.session, model.ActorTypeUser, test.userID, importItem.LocalID, &item))
	require.Equal(t, importItem.LocalURL, item.URL)
	require.Equal(t, test.actor()+"/pub/outbox/"+item.ActivityID.Hex(), item.URL)

	// The activity names this server's User, in its id, its actor and its addressing
	require.Equal(t, item.URL, item.Activity.GetString(vocab.PropertyID))
	require.Equal(t, test.actor(), item.Activity.GetString(vocab.PropertyActor))
	require.Equal(t, []string{test.followers(), "https://remote.example/@friend"}, item.Activity.GetSliceOfString(vocab.PropertyCC))
	require.Equal(t, []string{test.actor()}, item.Activity.GetSliceOfString(vocab.PropertyBCC))
	require.Contains(t, []string(item.Recipients), test.followers())
	require.NotContains(t, []string(item.Recipients), "https://old.example/@111111111111111111111111/pub/followers")

	// The row is described from the activity, not from what the document claimed
	require.Equal(t, vocab.ActivityTypeCreate, item.ActivityType)
	require.Equal(t, "https://old.example/notes/1", item.ObjectID, "an object that was not imported stays as exported")
	require.Equal(t, model.NewAnonymousPermissions(), item.Permissions)

	// It keeps its original date, and nothing else from the old journal
	require.Equal(t, int64(1_600_000_000_000), item.CreateDate)
	require.Zero(t, item.DeleteDate)

	// RULE: An import delivers nothing
	require.Empty(t, test.tasks.Drain())
}

// TestOutbox2_Import_EmbedsImportedStream verifies that an activity about a Stream this import
// brought over embeds the Stream's copy on this server, and that another User's import is ignored
func TestOutbox2_Import_EmbedsImportedStream(t *testing.T) {

	const remoteObject = "https://old.example/notes/2"
	const localObject = "https://example.com/notes/local"

	localJSONLD := mapof.Any{vocab.PropertyID: localObject, vocab.PropertyType: vocab.ObjectTypeNote, vocab.PropertyContent: "current"}

	t.Run("own import", func(t *testing.T) {

		test := newOutbox2ImportTest(t)
		test.objects[remoteObject] = importedObject{userID: test.userID, jsonLD: localJSONLD, publishDate: 5}
		importItem := model.NewImportItem()

		require.NoError(t, test.service.Import(test.session, nil, &importItem, &test.user, exported(t, remoteItem(remoteObject))))

		item := model.NewOutboxItem()
		require.NoError(t, test.service.LoadByID(test.session, model.ActorTypeUser, test.userID, importItem.LocalID, &item))
		require.Equal(t, localObject, item.ObjectID)
		require.Equal(t, "current", streams.NewDocument(item.Activity).Object().Content())
		require.Equal(t, int64(1_600_000_000_000), item.CreateDate, "an exported date wins over the Stream's")
	})

	t.Run("another User's import", func(t *testing.T) {

		test := newOutbox2ImportTest(t)
		test.objects[remoteObject] = importedObject{userID: primitive.NewObjectID(), jsonLD: localJSONLD}
		importItem := model.NewImportItem()

		require.NoError(t, test.service.Import(test.session, nil, &importItem, &test.user, exported(t, remoteItem(remoteObject))))

		item := model.NewOutboxItem()
		require.NoError(t, test.service.LoadByID(test.session, model.ActorTypeUser, test.userID, importItem.LocalID, &item))
		require.Equal(t, remoteObject, item.ObjectID)
	})
}

// TestOutbox2_Import_Refuses verifies that a document for another actor type, or one that is not JSON, stores nothing
func TestOutbox2_Import_Refuses(t *testing.T) {

	test := newOutbox2ImportTest(t)
	importItem := model.NewImportItem()

	stream := remoteItem("https://old.example/notes/3")
	stream.ActorType = model.ActorTypeStream

	require.Error(t, test.service.Import(test.session, nil, &importItem, &test.user, exported(t, stream)))
	require.Error(t, test.service.Import(test.session, nil, &importItem, &test.user, []byte("not json")))
	require.Zero(t, countOutbox2(t, test.session))
}

// TestOutbox2_UndoImport verifies that undoing removes the imported activity, only from its
// importer's outbox, and that undoing twice succeeds
func TestOutbox2_UndoImport(t *testing.T) {

	test := newOutbox2ImportTest(t)
	importItem := model.NewImportItem()
	importItem.UserID = test.userID

	require.NoError(t, test.service.Import(test.session, nil, &importItem, &test.user, exported(t, remoteItem("https://old.example/notes/4"))))
	require.Equal(t, int64(1), countOutbox2(t, test.session))

	// Another User's import record cannot remove this activity
	foreign := importItem
	foreign.UserID = primitive.NewObjectID()
	require.NoError(t, test.service.UndoImport(test.session, &foreign))
	require.Equal(t, int64(1), countOutbox2(t, test.session))

	require.NoError(t, test.service.UndoImport(test.session, &importItem))
	require.Zero(t, countOutbox2(t, test.session))

	require.NoError(t, test.service.UndoImport(test.session, &importItem))
}

// TestOutbox2_LegacyImporter verifies that an exported legacy OutboxMessage becomes an OutboxItem
// that embeds the imported Stream and takes its date
func TestOutbox2_LegacyImporter(t *testing.T) {

	const oldActor = "https://old.example/@111111111111111111111111"
	const remoteObject = "https://old.example/notes/5"
	const localObject = "https://example.com/notes/local"

	newMessage := func() model.OutboxMessage {
		message := model.NewOutboxMessage()
		message.ActorType = model.ActorTypeUser
		message.ActorID = primitive.NewObjectID()
		message.ActorURL = oldActor
		message.ActivityType = vocab.ActivityTypeCreate
		message.ObjectID = remoteObject
		message.Permissions = model.NewAnonymousPermissions()
		return message
	}

	exportedMessage := func(message model.OutboxMessage) []byte {
		document, err := json.Marshal(message)
		require.NoError(t, err)
		return document
	}

	t.Run("embeds the imported Stream", func(t *testing.T) {

		test := newOutbox2ImportTest(t)
		test.objects[remoteObject] = importedObject{
			userID:      test.userID,
			jsonLD:      mapof.Any{vocab.PropertyID: localObject, vocab.PropertyType: vocab.ObjectTypeNote},
			publishDate: 1_500_000_000_000,
		}

		message := newMessage()
		importItem := model.NewImportItem()
		importItem.UserID = test.userID

		importer := test.service.LegacyImporter()
		require.NoError(t, importer.Import(test.session, nil, &importItem, &test.user, exportedMessage(message)))
		require.Equal(t, message.OutboxMessageID, importItem.RemoteID)
		require.Equal(t, oldActor+"/pub/outbox/"+message.OutboxMessageID.Hex(), importItem.RemoteURL)

		item := model.NewOutboxItem()
		require.NoError(t, test.service.LoadByID(test.session, model.ActorTypeUser, test.userID, importItem.LocalID, &item))
		require.Equal(t, vocab.ActivityTypeCreate, item.ActivityType)
		require.Equal(t, localObject, item.ObjectID)
		require.Equal(t, test.actor(), item.Activity.GetString(vocab.PropertyActor))
		require.Equal(t, []string{vocab.NamespacePublic}, item.Activity.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, model.NewAnonymousPermissions(), item.Permissions)
		require.Equal(t, int64(1_500_000_000_000), item.CreateDate, "an export with no date takes the Stream's publish date")
		require.Empty(t, test.tasks.Drain())

		// The published object can be found again, as Stream.Delete needs
		published, err := test.service.HasObject(test.session, model.ActorTypeUser, test.userID, localObject)
		require.NoError(t, err)
		require.True(t, published)

		require.NoError(t, importer.UndoImport(test.session, &importItem))
		require.Zero(t, countOutbox2(t, test.session))
	})

	t.Run("keeps an exported date and an unknown object", func(t *testing.T) {

		test := newOutbox2ImportTest(t)
		message := newMessage()
		message.CreateDate = 1_400_000_000_000
		message.Permissions = model.Permissions{primitive.NewObjectID()}
		importItem := model.NewImportItem()

		require.NoError(t, test.service.LegacyImporter().Import(test.session, nil, &importItem, &test.user, exportedMessage(message)))

		item := model.NewOutboxItem()
		require.NoError(t, test.service.LoadByID(test.session, model.ActorTypeUser, test.userID, importItem.LocalID, &item))
		require.Equal(t, remoteObject, item.ObjectID)
		require.Equal(t, int64(1_400_000_000_000), item.CreateDate)
		require.Empty(t, item.Activity.GetSliceOfString(vocab.PropertyTo), "a restricted message names no one")
		require.Equal(t, message.Permissions, item.Permissions)
	})

	t.Run("refuses another actor type", func(t *testing.T) {

		test := newOutbox2ImportTest(t)
		message := newMessage()
		message.ActorType = model.ActorTypeStream
		importItem := model.NewImportItem()

		require.Error(t, test.service.LegacyImporter().Import(test.session, nil, &importItem, &test.user, exportedMessage(message)))
		require.Zero(t, countOutbox2(t, test.session))
	})
}

// TestImportedStreamFinder_Refuses verifies that only the User's own, finished Stream import is found
func TestImportedStreamFinder_Refuses(t *testing.T) {

	test := newOutbox2ImportTest(t)
	finder := importedStreamFinder{importItemService: &ImportItem{}, streamService: &Stream{}}

	save := func(remoteURL string, mutate func(*model.ImportItem)) {
		importItem := model.NewImportItem()
		importItem.UserID = test.userID
		importItem.Type = "emissary:stream"
		importItem.RemoteURL = remoteURL
		importItem.LocalURL = "https://example.com/local"
		importItem.StateID = model.ImportItemStateDone
		mutate(&importItem)
		require.NoError(t, test.session.Collection("ImportItem").Save(&importItem, "test"))
	}

	save("https://old.example/a", func(item *model.ImportItem) { item.UserID = primitive.NewObjectID() })
	save("https://old.example/b", func(item *model.ImportItem) { item.Type = "emissary:folder" })
	save("https://old.example/c", func(item *model.ImportItem) { item.LocalURL = "" })

	for _, remoteURL := range []string{"https://old.example/a", "https://old.example/b", "https://old.example/c", "https://old.example/unknown"} {
		object, _, found := finder.findImportedObject(test.session, test.userID, remoteURL)
		require.False(t, found, remoteURL)
		require.Nil(t, object, remoteURL)
	}
}

// TestImportItem_LoadNext_InCreationOrder verifies that items are imported in the order they were created
func TestImportItem_LoadNext_InCreationOrder(t *testing.T) {

	test := newOutbox2ImportTest(t)
	service := &ImportItem{}
	importID := primitive.NewObjectID()
	created := make([]primitive.ObjectID, 0, 20)

	for range 20 {
		importItem := model.NewImportItem()
		importItem.ImportID = importID
		importItem.UserID = test.userID
		importItem.StateID = model.ImportItemStateNew
		require.NoError(t, test.session.Collection("ImportItem").Save(&importItem, "test"))
		created = append(created, importItem.ImportItemID)
	}

	// Finish each item in turn, as the import task does, and read the next
	for _, expected := range created {
		next := model.NewImportItem()
		require.NoError(t, service.LoadNext(test.session, test.userID, importID, &next))
		require.Equal(t, expected, next.ImportItemID)

		next.StateID = model.ImportItemStateDone
		require.NoError(t, test.session.Collection("ImportItem").Save(&next, "done"))
	}
}
