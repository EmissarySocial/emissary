package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// existingStreamSource returns a record that has already been saved once
func existingStreamSource() model.StreamSource {
	result := validStreamSource()
	result.CreateDate = 1_700_000_000 // what IsNew() reads
	result.Status = model.StreamSourceStatusSuccess
	return result
}

/******************************************
 * When A Save Synchronizes
 ******************************************/

// TestStreamSource_SaveSyncsANewRecord pins the case that cannot be asked for: nothing polls, so a
// record that has never synced must fetch its first content on its own, or the Stream stays empty
// until somebody happens to push to the repository.
func TestStreamSource_SaveSyncsANewRecord(t *testing.T) {

	service, session := newStreamSourceService()
	streamSource := validStreamSource()

	require.False(t, streamSource.SyncNow, "a new record does not have to ASK")
	require.NoError(t, service.Save(session, &streamSource, "Created"))

	require.Len(t, session.publishedTasks(), 1)
	require.Equal(t, model.StreamSourceStatusLoading, streamSource.Status)
}

// TestStreamSource_SaveDoesNotSyncWithoutAsking confirms that editing an existing record leaves the
// remote source alone.  Rotating a webhook token or fixing a typo in a label must not refetch and
// republish a page, because Stream.Save federates and notifies.
func TestStreamSource_SaveDoesNotSyncWithoutAsking(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.Save(session, &streamSource, "Updated"))

	require.Empty(t, session.publishedTasks(), "an unasked save queues nothing")
	require.Equal(t, model.StreamSourceStatusSuccess, streamSource.Status, "Status is left alone")
}

// TestStreamSource_SaveSyncsWhenAsked pins the Sync Now button: set syncNow on the record, save it,
// and the queue does the rest
func TestStreamSource_SaveSyncsWhenAsked(t *testing.T) {

	streamSource := existingStreamSource()
	streamSource.SyncNow = true

	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.Save(session, &streamSource, "Sync Now"))

	tasks := session.publishedTasks()
	require.Len(t, tasks, 1)
	require.Equal(t, TaskSyncStreamSource, tasks[0].Name)
	require.Equal(t, streamSource.StreamSourceID.Hex(), tasks[0].Arguments.GetString("streamSourceId"))
}

// TestStreamSource_SaveShowsLoading confirms that a requested sync is visible before the worker
// runs.  A human pressed a button, and the worker cannot write LOADING itself -- its own write
// rolls back with the attempt that failed.
func TestStreamSource_SaveShowsLoading(t *testing.T) {

	streamSource := existingStreamSource()
	streamSource.SyncNow = true
	streamSource.StatusMessage = "Source not found"

	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.Save(session, &streamSource, "Sync Now"))

	require.Len(t, session.collection.saved, 1)
	saved := session.collection.saved[0]

	require.Equal(t, model.StreamSourceStatusLoading, saved.Status, "the status is STORED, not just set")
	require.Empty(t, saved.StatusMessage, "a stale failure must not sit beside a running sync")
	require.NotZero(t, saved.LastSynced)
}

/******************************************
 * SyncNow Is A Command, Not State
 ******************************************/

// TestStreamSource_SyncNowIsConsumed confirms that the flag is cleared once acted on.  A caller
// that reused the value -- a pipeline that saves twice, a retry -- would otherwise queue a second
// sync nobody asked for.
func TestStreamSource_SyncNowIsConsumed(t *testing.T) {

	streamSource := existingStreamSource()
	streamSource.SyncNow = true

	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.Save(session, &streamSource, "Sync Now"))
	require.False(t, streamSource.SyncNow, "the command has been carried out")

	require.Len(t, session.publishedTasks(), 1)

	// Saving the same value again asks for nothing
	require.NoError(t, service.Save(session, &streamSource, "Updated"))
	require.Empty(t, session.publishedTasks())
}

// TestStreamSource_SyncNowIsNeverStored pins the bson tag.  A stored syncNow would make every
// record that was ever synced by hand re-sync on its next save, forever.
func TestStreamSource_SyncNowIsNeverStored(t *testing.T) {

	streamSource := existingStreamSource()
	streamSource.SyncNow = true

	encoded, err := bson.Marshal(streamSource)
	require.NoError(t, err)

	var stored bson.M
	require.NoError(t, bson.Unmarshal(encoded, &stored))

	require.NotContains(t, stored, "syncNow")
	require.Contains(t, stored, "url", "the rest of the record still persists")
}

// TestStreamSource_SyncNowIsNeverExported confirms that the flag stays out of JSON, where it would
// read as a field somebody could set on a record they are merely reading
func TestStreamSource_SyncNowIsNeverExported(t *testing.T) {

	streamSource := existingStreamSource()
	streamSource.SyncNow = true

	encoded, err := bson.MarshalExtJSON(streamSource, false, false)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "syncNow")
}

/******************************************
 * The Form Sets It
 ******************************************/

// TestStreamSource_SyncNowIsSettableBySchema confirms that a Template form can set this value.  The
// with-stream-source pipeline sets it through `set-data`, which writes through the schema, so a
// field missing from GetPointer would silently ignore the Sync Now button.
func TestStreamSource_SyncNowIsSettableBySchema(t *testing.T) {

	service, _ := newStreamSourceService()
	streamSource := validStreamSource()

	require.NoError(t, service.Schema().Set(&streamSource, "syncNow", true))
	require.True(t, streamSource.SyncNow)

	require.NoError(t, service.Schema().Set(&streamSource, "syncNow", false))
	require.False(t, streamSource.SyncNow)
}

// TestStreamSource_SyncNowValidates confirms that a record carrying the flag passes its own schema.
// A property missing from the schema is REMOVED by Normalize rather than refused, so a Sync Now
// that silently did nothing is the failure this rules out.
func TestStreamSource_SyncNowValidates(t *testing.T) {

	service, session := newStreamSourceService()
	streamSource := validStreamSource()
	streamSource.SyncNow = true

	require.NoError(t, service.Save(session, &streamSource, "Created"))
}

/******************************************
 * ModelService Interface
 ******************************************/

// TestStreamSource_ModelService confirms that the service satisfies the interface the
// with-stream-source step needs.  Nothing checks this at compile time from the step's side:
// Factory.ModelService returns an interface, so a missing method is a build error here and a
// nil service -- a 500 on the settings screen -- if the type switch is what was forgotten.
func TestStreamSource_ModelService(t *testing.T) {

	var service ModelService = &StreamSource{}

	require.Equal(t, "StreamSource", service.ObjectType())

	created, isStreamSource := service.ObjectNew().(*model.StreamSource)
	require.True(t, isStreamSource, "ObjectNew must return a *model.StreamSource")
	require.NotZero(t, created.StreamSourceID)
	require.NotEmpty(t, created.WebhookToken(), "ObjectNew must produce a usable record")
}

// TestStreamSource_ObjectID reads the primary key, and answers zero for anything else
func TestStreamSource_ObjectID(t *testing.T) {

	service, _ := newStreamSourceService()
	streamSource := validStreamSource()

	require.Equal(t, streamSource.StreamSourceID, service.ObjectID(&streamSource))

	// A different model reaching this service is a wiring defect, not a record to guess at
	other := model.NewStream()
	require.True(t, service.ObjectID(&other).IsZero())
}

// TestStreamSource_ObjectSaveAndDelete confirms that each verb routes to its OWN method.  These are
// thin delegations, and a copy-paste that pointed ObjectDelete at Save would compile, pass a type
// check, and quietly re-save the record that somebody asked to remove.
func TestStreamSource_ObjectSaveAndDelete(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.ObjectSave(session, &streamSource, "Updated"))
	require.Len(t, session.collection.saved, 1)
	require.Zero(t, session.collection.records[0].DeleteDate, "a save must not delete")

	require.NoError(t, service.ObjectDelete(session, &streamSource, "Removed"))
	require.NotZero(t, session.collection.records[0].DeleteDate, "a delete must actually delete")
}

// TestStreamSource_ObjectLoad returns the record as a data.Object
func TestStreamSource_ObjectLoad(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)

	loaded, err := service.ObjectLoad(session, exp.Equal("_id", streamSource.StreamSourceID))

	require.NoError(t, err)
	require.Equal(t, streamSource.StreamSourceID.Hex(), loaded.ID())
}

// TestStreamSource_ObjectWrongType confirms that a mismatched object is refused rather than
// silently ignored.  Factory.ModelService dispatches on type, so reaching here with the wrong one
// means the type switch and this service disagree.
func TestStreamSource_ObjectWrongType(t *testing.T) {

	service, session := newStreamSourceService()
	stream := model.NewStream()

	require.Error(t, service.ObjectSave(session, &stream, "Updated"))
	require.Error(t, service.ObjectDelete(session, &stream, "Removed"))
	require.Empty(t, session.collection.saved, "a refused object writes nothing")
}

// TestStreamSource_ObjectUserCan pins that a StreamSource can never be addressed on its own.  It is
// reached only through the Stream it populates, whose action was already checked.
func TestStreamSource_ObjectUserCan(t *testing.T) {

	service, _ := newStreamSourceService()
	streamSource := validStreamSource()

	err := service.ObjectUserCan(&streamSource, model.NewAuthorization(), "edit")

	require.Error(t, err)
	require.True(t, derp.IsUnauthorized(err), "got %v", err)
}

// TestStreamSource_AccessLister confirms that the record grants no access of its own.  It has no
// owner -- it belongs to a Stream -- so "author" must name nobody, or a Template action carrying
// that role would hand the record to whoever asked.
func TestStreamSource_AccessLister(t *testing.T) {

	var accessLister model.AccessLister = &model.StreamSource{}

	require.Equal(t, "default", accessLister.State(), "Template actions declare their roles under this state")
	require.False(t, accessLister.IsAuthor(primitive.NewObjectID()))
	require.False(t, accessLister.IsMyself(primitive.NewObjectID()))
	require.Empty(t, accessLister.RolesToGroupIDs("author", "myself"), "a StreamSource has no owner")
	require.Empty(t, accessLister.RolesToPrivilegeIDs("author"))
}
