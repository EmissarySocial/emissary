package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// existingStreamSource returns a record that has already been saved and synced once
func existingStreamSource() model.StreamSource {
	result := validStreamSource()
	result.CreateDate = 1_700_000_000
	result.Status = model.StreamSourceStatusSuccess
	return result
}

/******************************************
 * Saving Reaches The Network
 ******************************************/

// TestStreamSource_SaveAlwaysSyncs pins the decision that a save IS the request to synchronize.
// Nothing polls, so a save is the only moment a human tells Emissary this record is worth reading
// -- and a repeat costs one conditional GET that answers 304.
func TestStreamSource_SaveAlwaysSyncs(t *testing.T) {

	saves := map[string]model.StreamSource{
		"a brand new record": validStreamSource(),
		"an existing record": existingStreamSource(),
	}

	for name, streamSource := range saves {
		t.Run(name, func(t *testing.T) {

			service, session := newStreamSourceService(streamSource)

			require.NoError(t, service.Save(session, &streamSource, "Saved"))

			tasks := session.publishedTasksNamed(TaskSyncStreamSourceNow)
			require.Len(t, tasks, 1, "every save queues exactly one sync")
			require.Empty(t, tasks[0].Signature, "a signed task can never run immediately")
			require.Equal(t, streamSource.StreamSourceID.Hex(), tasks[0].Arguments.GetString("streamSourceId"))
		})
	}
}

// TestStreamSource_WebhookSyncIsDeduplicated confirms that two pings cannot double the work.  The
// signature collapses a second task into the one already queued for this record.
func TestStreamSource_WebhookSyncIsDeduplicated(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)

	service.PublishSyncTask(session, streamSource.StreamSourceID)
	service.PublishSyncTask(session, streamSource.StreamSourceID)

	tasks := session.publishedTasksNamed(TaskSyncStreamSource)
	require.Len(t, tasks, 2, "the spool holds both")

	require.Equal(t, tasks[0].Signature, tasks[1].Signature,
		"a shared signature is what collapses them at the queue")
	require.Equal(t, "StreamSource-Sync:"+streamSource.StreamSourceID.Hex(), tasks[0].Signature)
}

// TestStreamSource_SyncNowIsNotDeduplicated pins the trade this design accepts.  The interactive
// task carries NO signature, because turbine refuses to run a signed task from memory at any
// priority -- so two quick presses really do queue two syncs.  Each costs one conditional GET
// that answers 304, which is why that is affordable.
func TestStreamSource_SyncNowIsNotDeduplicated(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.Save(session, &streamSource, "Saved"))
	require.NoError(t, service.Save(session, &streamSource, "Saved again"))

	tasks := session.publishedTasksNamed(TaskSyncStreamSourceNow)
	require.Len(t, tasks, 2)

	for _, task := range tasks {
		require.Empty(t, task.Signature, "a signature would send this to storage and the 1-minute poller")
	}
}

// TestStreamSource_SaveShowsLoading confirms that a sync is visible before the worker runs.  The
// worker cannot write LOADING itself: its own write rolls back with the attempt that failed.
func TestStreamSource_SaveShowsLoading(t *testing.T) {

	streamSource := existingStreamSource()
	streamSource.StatusMessage = "Source not found"

	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.Save(session, &streamSource, "Saved"))

	require.Len(t, session.collection.saved, 1)
	saved := session.collection.saved[0]

	require.Equal(t, model.StreamSourceStatusLoading, saved.Status, "the status is STORED, not just set")
	require.Empty(t, saved.StatusMessage, "a stale failure must not sit beside a running sync")
	require.NotZero(t, saved.LastSynced)
}

// TestStreamSource_BookkeepingNeverSyncs pins the separation that keeps a sync from feeding itself.
// The status writers and the sync's own save go through service.save, so a running sync cannot
// queue another one -- the signature would not stop it, because the first task has already left
// the queue by the time its handler saves.  They DO publish an SSE nudge, which is why this
// counts sync tasks rather than tasks.
func TestStreamSource_BookkeepingNeverSyncs(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.SetStatusLoading(session, &streamSource))
	require.NoError(t, service.SetStatusSuccess(session, &streamSource))
	require.NoError(t, service.SetStatusFailure(session, &streamSource, "Source not found"))
	require.NoError(t, service.SetStatusMessage(session, &streamSource, "Retrying"))

	require.Empty(t, session.publishedTasksNamed(TaskSyncStreamSource), "bookkeeping is not a request to read the source")
	require.Empty(t, session.publishedTasksNamed(TaskSyncStreamSourceNow), "..and not a request to read it right now, either")
}

/******************************************
 * ModelService Interface
 ******************************************/

// TestStreamSource_ModelService confirms that the service satisfies the interface the
// with-stream-source step needs.  Nothing checks this from the step's side: Factory.ModelService
// returns an interface, so a missing method is a build error here and a nil service -- a 500 on
// the settings screen -- if the type switch is what was forgotten.
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

// TestStreamSource_ObjectSaveSyncs confirms that the generic `save` step reaches the same Save the
// settings form does.  A delegation that wrote through the collection instead would store the
// record and never read the source.
func TestStreamSource_ObjectSaveSyncs(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.ObjectSave(session, &streamSource, "Updated"))
	require.Len(t, session.publishedTasksNamed(TaskSyncStreamSourceNow), 1)
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

/******************************************
 * Realtime (SSE) Nudges
 ******************************************/

// TestStreamSource_EveryWriteNudgesTheStream pins the reason `service.save` exists.  The settings
// screen shows Status and the time of the last check, and a synchronization finishes in the
// BACKGROUND minutes after Sync Now returns -- so a write the screen never hears about leaves
// those two fields stale until somebody reloads, with nothing reporting it.
//
// Every write path is listed here on purpose.  A fifth status writer added later would be caught
// by this test only if it is added to the list, so the point of the list is to be the place that
// says what "every write" means.
//
// Delete is deliberately NOT in the list.  Its only caller, the `delete-source` action, ends with
// `refresh-page`, so the browser that pressed Stop Syncing refetches either way.  What that gives
// up is a SECOND browser open on the same article, which will keep showing a source that is gone
// until something else refreshes it.
func TestStreamSource_EveryWriteNudgesTheStream(t *testing.T) {

	writes := map[string]func(*StreamSource, streamSourceSession, *model.StreamSource) error{

		"Save": func(service *StreamSource, session streamSourceSession, streamSource *model.StreamSource) error {
			return service.Save(session, streamSource, "Saved")
		},
		"SetStatusLoading": func(service *StreamSource, session streamSourceSession, streamSource *model.StreamSource) error {
			return service.SetStatusLoading(session, streamSource)
		},
		"SetStatusSuccess": func(service *StreamSource, session streamSourceSession, streamSource *model.StreamSource) error {
			return service.SetStatusSuccess(session, streamSource)
		},
		"SetStatusFailure": func(service *StreamSource, session streamSourceSession, streamSource *model.StreamSource) error {
			return service.SetStatusFailure(session, streamSource, "Source not found")
		},
		"SetStatusMessage": func(service *StreamSource, session streamSourceSession, streamSource *model.StreamSource) error {
			return service.SetStatusMessage(session, streamSource, "Retrying")
		},
		"saveSyncState": func(service *StreamSource, session streamSourceSession, streamSource *model.StreamSource) error {
			return service.saveSyncState(session, streamSource, "Checked: unchanged")
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {

			streamSource := existingStreamSource()
			service, session := newStreamSourceService(streamSource)

			require.NoError(t, write(service, session, &streamSource))

			tasks := session.publishedTasksNamed("PublishRealtimeMessage")
			require.Len(t, tasks, 1, "%s published no realtime nudge", name)

			// RULE: Addressed by the STREAM's id.  A StreamSource has no page and no SSE route of
			// its own, so a nudge sent to its own id would reach nobody -- and reach them silently.
			require.Equal(t, streamSource.StreamID.Hex(), tasks[0].Arguments.GetString("objectId"))
			require.Equal(t, realtime.TopicStreamSourceUpdated, tasks[0].Arguments.GetInt("topic"))
		})
	}
}

// TestStreamSource_FailedWriteNudgesNobody confirms that a nudge follows the write rather than the
// attempt.  A browser told to refetch after a failed save would re-render the same record it
// already had, and read that as the change having been applied.
func TestStreamSource_FailedWriteNudgesNobody(t *testing.T) {

	streamSource := existingStreamSource()
	service, session := newStreamSourceService(streamSource)
	session.collection.saveError = derp.Internal("test", "collection is offline")

	require.Error(t, service.SetStatusSuccess(session, &streamSource))
	require.Empty(t, session.publishedTasksNamed("PublishRealtimeMessage"))
}
