package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// attachmentTask returns a well-formed SyncStreamSourceAttachment task
func attachmentTask() queue.Task {
	return queue.Task{
		Name: service.TaskSyncStreamSourceAttachment,
		Arguments: mapof.Any{
			"hostname":     "example.com",
			"streamId":     "62f8f1d4d7f1c8e6b4a1c2d3",
			"attachmentId": "62f8f1d4d7f1c8e6b4a1c2d4",
		},
	}
}

// TestSyncStreamSourceAttachment_MalformedIDs confirms that an unparseable identifier is a
// permanent failure.  The nil factory is the proof that the guards run before anything is loaded.
func TestSyncStreamSourceAttachment_MalformedIDs(t *testing.T) {

	for _, name := range []string{"streamId", "attachmentId"} {

		args := attachmentTask().Arguments
		args[name] = "not-an-id"

		result := SyncStreamSourceAttachment(nil, args)
		require.Equal(t, queue.ResultStatusFailure, result.Status, "argument %q", name)
	}
}

// TestSyncStreamSourceAttachment_IsRegistered confirms that the task reaches a handler.  A name that
// falls through the switch returns Ignored, and the file is never downloaded.
func TestSyncStreamSourceAttachment_IsRegistered(t *testing.T) {

	result := New(unresolvableFactory{}).Run(attachmentTask())

	require.NotEqual(t, queue.ResultStatusIgnored, result.Status, "the task reaches no handler")
	require.Equal(t, queue.ResultStatusError, result.Status, "an unknown hostname may become known later")
}

// TestSyncStreamSourceAttachment_Priority confirms that a download runs at the same priority as the
// sync that queued it
func TestSyncStreamSourceAttachment_Priority(t *testing.T) {

	task := attachmentTask()
	task.Priority = -1

	require.NoError(t, PreProcessor(&task))

	// RULE: Assert the EXACT value.  A name that matches no case leaves Priority at the -1 sentinel.
	require.Equal(t, 16, task.Priority)
}

// TestSyncStreamSourceAttachment_HooksIgnoreIt confirms that the StreamSource status hooks do not
// act on a download.  A download writes only its own attachment, never the StreamSource record.
func TestSyncStreamSourceAttachment_HooksIgnoreIt(t *testing.T) {

	consumer := New(unresolvableFactory{})
	task := attachmentTask()

	require.NoError(t, consumer.OnSuccess(task))
	require.NoError(t, consumer.OnError(task, nil))
	require.NoError(t, consumer.OnFailure(task, nil))
}
