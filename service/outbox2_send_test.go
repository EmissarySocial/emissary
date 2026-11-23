package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// requireOneFanOut asserts that exactly one fan-out task was spooled, for the given actor, and returns the activity it carries
func requireOneFanOut(t *testing.T, spool *postcommit.Tasks, actorURL string) mapof.Any {

	t.Helper()

	tasks := spool.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, delivery.TaskFanOut, tasks[0].Name, "every send goes through hannibal's fan-out")
	require.Equal(t, "example.com", tasks[0].Arguments.GetString("hostname"))
	require.Equal(t, actorURL, tasks[0].Arguments.GetString("actor"))

	return tasks[0].Arguments.GetMap("activity")
}

// TestOutbox2_SendAccept verifies that an Accept embeds the Follow and is addressed to the Follow's actor
func TestOutbox2_SendAccept(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	follow := streams.NewDocument(mapof.Any{
		vocab.PropertyID:     "https://remote.example/follows/1",
		vocab.PropertyType:   vocab.ActivityTypeFollow,
		vocab.PropertyActor:  "https://remote.example/@bob",
		vocab.PropertyObject: "https://example.com/@alice",
	})

	require.NoError(t, outbox2.SendAccept(session, "https://example.com/@alice", "https://example.com/@alice/pub/followers/1", follow))

	activity := requireOneFanOut(t, spool, "https://example.com/@alice")
	require.Equal(t, "https://example.com/@alice/pub/followers/1", activity.GetString(vocab.PropertyID))
	require.Equal(t, vocab.ActivityTypeAccept, activity.GetString(vocab.PropertyType))
	require.Equal(t, []string{"https://remote.example/@bob"}, activity.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, "https://remote.example/follows/1", activity.GetMap(vocab.PropertyObject).GetString(vocab.PropertyID))
}

// TestOutbox2_SendAccept_EmbeddedActor verifies that an Accept reaches a Follow's actor when the Follow embeds it, without loading it
func TestOutbox2_SendAccept_EmbeddedActor(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	follow := streams.NewDocument(mapof.Any{
		vocab.PropertyType:   vocab.ActivityTypeFollow,
		vocab.PropertyActor:  mapof.Any{vocab.PropertyID: "https://remote.example/@bob", vocab.PropertyType: vocab.ActorTypePerson},
		vocab.PropertyObject: "https://example.com/@alice",
	})

	require.NoError(t, outbox2.SendAccept(session, "https://example.com/@alice", "https://example.com/@alice/pub/followers/1", follow))

	activity := requireOneFanOut(t, spool, "https://example.com/@alice")
	require.Equal(t, []string{"https://remote.example/@bob"}, activity.GetSliceOfString(vocab.PropertyTo))
}

// TestOutbox2_SendRejectFollow verifies that a Reject describes the Follow by its actor and object, and is addressed to the follower
func TestOutbox2_SendRejectFollow(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	require.NoError(t, outbox2.SendRejectFollow(session, "https://example.com/@alice", "https://example.com/@alice/pub/followers/1#reject", "https://remote.example/@bob"))

	activity := requireOneFanOut(t, spool, "https://example.com/@alice")
	require.Equal(t, vocab.ActivityTypeReject, activity.GetString(vocab.PropertyType))
	require.Equal(t, []string{"https://remote.example/@bob"}, activity.GetSliceOfString(vocab.PropertyTo))

	follow := activity.GetMap(vocab.PropertyObject)
	require.Equal(t, vocab.ActivityTypeFollow, follow.GetString(vocab.PropertyType))
	require.Equal(t, "https://remote.example/@bob", follow.GetString(vocab.PropertyActor))
	require.Equal(t, "https://example.com/@alice", follow.GetString(vocab.PropertyObject))
}

// TestOutbox2_SendFollow verifies that a Follow names the followed actor as its object and its only recipient
func TestOutbox2_SendFollow(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	require.NoError(t, outbox2.SendFollow(session, "https://example.com/@alice", "https://example.com/@alice/pub/following/1", "https://remote.example/@bob"))

	activity := requireOneFanOut(t, spool, "https://example.com/@alice")
	require.Equal(t, "https://example.com/@alice/pub/following/1", activity.GetString(vocab.PropertyID))
	require.Equal(t, vocab.ActivityTypeFollow, activity.GetString(vocab.PropertyType))
	require.Equal(t, "https://remote.example/@bob", activity.GetString(vocab.PropertyObject))
	require.Equal(t, []string{"https://remote.example/@bob"}, activity.GetSliceOfString(vocab.PropertyTo))
	require.NotEmpty(t, activity.GetString(vocab.PropertyPublished))
}

// TestOutbox2_SendUndoFollow verifies that an Undo embeds the original Follow and is addressed to the actor that was followed
func TestOutbox2_SendUndoFollow(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	follow := mapof.Any{
		vocab.PropertyID:     "https://example.com/@alice/pub/following/1",
		vocab.PropertyType:   vocab.ActivityTypeFollow,
		vocab.PropertyActor:  "https://example.com/@alice",
		vocab.PropertyObject: "https://remote.example/@bob",
	}

	require.NoError(t, outbox2.SendUndoFollow(session, "https://example.com/@alice", follow))

	activity := requireOneFanOut(t, spool, "https://example.com/@alice")
	require.Equal(t, vocab.ActivityTypeUndo, activity.GetString(vocab.PropertyType))
	require.Equal(t, []string{"https://remote.example/@bob"}, activity.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, "https://example.com/@alice/pub/following/1", activity.GetMap(vocab.PropertyObject).GetString(vocab.PropertyID))
}

// TestOutbox2_SendUndoFollow_ObjectShapes verifies that an Undo reaches the followed actor whether the
// Follow names it by URL or embeds it, and reaches no one when the Follow names no one
func TestOutbox2_SendUndoFollow_ObjectShapes(t *testing.T) {

	// test sends an Undo of a Follow whose object is the given value, and checks its recipients
	test := func(name string, object any, expected []string) {
		t.Run(name, func(t *testing.T) {

			outbox2, session, spool := newOutbox2PublishTest(t)
			follow := mapof.Any{vocab.PropertyType: vocab.ActivityTypeFollow, vocab.PropertyObject: object}

			require.NoError(t, outbox2.SendUndoFollow(session, "https://example.com/@alice", follow))

			activity := requireOneFanOut(t, spool, "https://example.com/@alice")
			require.Equal(t, expected, activity.GetSliceOfString(vocab.PropertyTo))
		})
	}

	test("a URL", "https://remote.example/@bob", []string{"https://remote.example/@bob"})
	test("an embedded actor", mapof.Any{vocab.PropertyID: "https://remote.example/@bob", vocab.PropertyType: vocab.ActorTypePerson}, []string{"https://remote.example/@bob"})
	test("an empty URL", "", []string{})
}

// TestOutbox2_SendAnnounce verifies that a Stream's boost is addressed to Public and its followers, never to the post's own audience
func TestOutbox2_SendAnnounce(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	const streamURL = "https://example.com/000000000000000000000001"

	post := streams.NewDocument(mapof.Any{
		vocab.PropertyID:      "https://remote.example/notes/1",
		vocab.PropertyType:    vocab.ObjectTypeNote,
		vocab.PropertyTo:      []any{"https://remote.example/@carol"},
		vocab.PropertyContent: "Hello",
	})

	require.NoError(t, outbox2.SendAnnounce(session, streamURL, streamURL+"/announced/1", post, streamURL+"/pub/followers"))

	activity := requireOneFanOut(t, spool, streamURL)
	require.Equal(t, streamURL+"/announced/1", activity.GetString(vocab.PropertyID))
	require.Equal(t, vocab.ActivityTypeAnnounce, activity.GetString(vocab.PropertyType))
	require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, activity.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, []string{streamURL + "/pub/followers"}, activity.GetSliceOfString(vocab.PropertyCC))
	require.Equal(t, "https://remote.example/notes/1", activity.GetMap(vocab.PropertyObject).GetString(vocab.PropertyID))
}

// TestOutbox2_Send_RequireActor verifies that every send refuses to queue an activity with no actor
func TestOutbox2_Send_RequireActor(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)
	document := streams.NewDocument(mapof.Any{vocab.PropertyActor: "https://remote.example/@bob"})

	require.Error(t, outbox2.SendAccept(session, "", "accept", document))
	require.Error(t, outbox2.SendRejectFollow(session, "", "reject", "https://remote.example/@bob"))
	require.Error(t, outbox2.SendFollow(session, "", "follow", "https://remote.example/@bob"))
	require.Error(t, outbox2.SendUndoFollow(session, "", mapof.Any{vocab.PropertyObject: "https://remote.example/@bob"}))
	require.Error(t, outbox2.SendAnnounce(session, "", "announce", document, "https://example.com/1/pub/followers"))
	require.Empty(t, spool.Drain(), "nothing may be queued without an actor")
}
