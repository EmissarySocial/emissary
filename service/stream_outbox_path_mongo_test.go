package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/queries/upgrades"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests follow a Stream along the whole path from a save to what followers are sent,
// against a real MongoDB.  Each joins steps that the other outbox tests check one at a time.

// syncActivities runs every outbox task that is queued, and returns the activities that were sent
func (test outboxSyncTest) syncActivities(t *testing.T) []mapof.Any {

	t.Helper()

	result := make([]mapof.Any, 0)

	for _, task := range test.tasks.Drain() {

		switch task.Name {

		case StreamOutboxTaskName:
			streamID, err := primitive.ObjectIDFromHex(task.Arguments.GetString("streamId"))
			require.NoError(t, err)
			require.NoError(t, test.service.SyncOutbox(test.session, streamID, OutboxChanges{}))
			result = append(result, test.syncActivities(t)...)

		case delivery.TaskFanOut:
			result = append(result, task.Arguments.GetMap("activity"))
		}
	}

	return result
}

// outboxRows returns the number of activities of one type that the outbox holds for a Stream
func (test outboxSyncTest) outboxRows(t *testing.T, stream *model.Stream, activityType string) int64 {

	t.Helper()

	criteria := exp.Equal("objectId", stream.ActivityPubURL()).AndEqual("activityType", activityType)
	count, err := test.session.Collection("OutboxItem").Count(criteria)
	require.NoError(t, err)

	return count
}

// TestStreamOutboxPath_OnePipelineOneCreate confirms that a request which saves a Stream, adds
// an attachment, and saves it again sends one Create, and that the Create carries the attachment
func TestStreamOutboxPath_OnePipelineOneCreate(t *testing.T) {

	test := newOutboxSyncTest(t)

	// The first save of the request puts the Stream in the outbox
	stream := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &stream, "created"))

	// A later step of the same request uploads a file
	attachment := model.NewAttachment(model.AttachmentObjectTypeStream, stream.StreamID)
	attachment.URL = "https://example.test/" + stream.StreamID.Hex() + "/attachments/" + attachment.AttachmentID.Hex()
	attachment.Label = "Cover art"
	require.NoError(t, test.session.Collection("Attachment").Save(&attachment, "uploaded"))

	// And its last step saves again
	stream.Label = "The Final Cut"
	require.NoError(t, test.service.Save(test.session, &stream, "saved again"))

	// Nothing is sent until the request commits, and then both tasks run
	activities := test.syncActivities(t)
	require.Len(t, activities, 1, "two saves in one request send one activity")
	require.Equal(t, vocab.ActivityTypeCreate, activities[0].GetString(vocab.PropertyType))

	object := activities[0].GetMap(vocab.PropertyObject)
	require.Equal(t, "The Final Cut", object.GetString(vocab.PropertyName))

	attachments := object.GetSliceOfMap(vocab.PropertyAttachment)
	require.Len(t, attachments, 1)
	require.Equal(t, attachment.URL, attachments[0].GetString(vocab.PropertyURL))

	require.Equal(t, int64(1), test.outboxRows(t, &stream, vocab.ActivityTypeCreate))
}

// TestStreamOutboxPath_FingerprintIsStable confirms that one Stream gives one fingerprint,
// from the copy that was saved and from a copy read back out of the database
func TestStreamOutboxPath_FingerprintIsStable(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := test.newPublicStream("published")
	stream.Content.HTML = "<p>There is no fate but what we make.</p>"
	require.NoError(t, test.service.Save(test.session, &stream, "published"))

	_, first, err := test.service.outboxObject(test.session, &stream)
	require.NoError(t, err)

	_, second, err := test.service.outboxObject(test.session, &stream)
	require.NoError(t, err)
	require.Equal(t, first, second)

	loaded := storedStream(t, test.service, test.session, stream.StreamID)
	_, reloaded, err := test.service.outboxObject(test.session, &loaded)
	require.NoError(t, err)
	require.Equal(t, first, reloaded)

	// The task records the same value
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))
	require.Equal(t, first, test.fingerprint(t, stream.StreamID))
}

// TestStreamOutboxPath_FingerprintFields pins which properties of a Stream take part in its
// fingerprint.  A property that followers never see must not, or every counter refresh would
// send an Update.  Update this list when the object that followers are sent gains a property.
func TestStreamOutboxPath_FingerprintFields(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := test.newPublicStream("published")
	require.NoError(t, test.service.Save(test.session, &stream, "published"))

	_, baseline, err := test.service.outboxObject(test.session, &stream)
	require.NoError(t, err)

	for name, testCase := range map[string]struct {
		edit    func(stream *model.Stream)
		changes bool
	}{
		// Followers are sent these
		"label":        {changes: true, edit: func(stream *model.Stream) { stream.Label = "No Fate" }},
		"content":      {changes: true, edit: func(stream *model.Stream) { stream.Content.HTML = "<p>Edited</p>" }},
		"context":      {changes: true, edit: func(stream *model.Stream) { stream.Context = "https://example.test/context" }},
		"inReplyTo":    {changes: true, edit: func(stream *model.Stream) { stream.InReplyTo = "https://example.test/parent" }},
		"publishDate":  {changes: true, edit: func(stream *model.Stream) { stream.PublishDate++ }},
		"replyPolicy":  {changes: true, edit: func(stream *model.Stream) { stream.ReplyPolicy = model.ReplyPolicyNobody }},
		"defaultAllow": {changes: true, edit: func(stream *model.Stream) { stream.DefaultAllow = model.NewPermissions() }},
		"attributedTo": {changes: true, edit: func(stream *model.Stream) { stream.AttributedTo.ProfileURL = "https://example.test/@other" }},

		// Followers are never sent these
		"summary":      {edit: func(stream *model.Stream) { stream.Summary = "Edited" }},
		"state":        {edit: func(stream *model.Stream) { stream.StateID.Set("featured") }},
		"rank":         {edit: func(stream *model.Stream) { stream.Rank++ }},
		"shuffle":      {edit: func(stream *model.Stream) { stream.Shuffle++ }},
		"isFeatured":   {edit: func(stream *model.Stream) { stream.IsFeatured = true }},
		"replyCount":   {edit: func(stream *model.Stream) { stream.ReplyCount++ }},
		"likeCount":    {edit: func(stream *model.Stream) { stream.LikeCount++ }},
		"dislikeCount": {edit: func(stream *model.Stream) { stream.DislikeCount++ }},
		"shareCount":   {edit: func(stream *model.Stream) { stream.ShareCount++ }},
		"syndication":  {edit: func(stream *model.Stream) { stream.Syndication.Set([]string{"bandwagon"}) }},
		"fingerprint":  {edit: func(stream *model.Stream) { stream.Fingerprint.Set("abc123") }},
	} {
		t.Run(name, func(t *testing.T) {

			edited := storedStream(t, test.service, test.session, stream.StreamID)
			testCase.edit(&edited)

			_, fingerprint, err := test.service.outboxObject(test.session, &edited)
			require.NoError(t, err)

			if testCase.changes {
				require.NotEqual(t, baseline, fingerprint)
				return
			}

			require.Equal(t, baseline, fingerprint)
		})
	}
}

// TestStreamOutboxPath_FailedSendRollsBack confirms that a task whose transaction fails leaves
// no claim and no outbox row behind, so that the retry sends the Create
func TestStreamOutboxPath_FailedSendRollsBack(t *testing.T) {

	database := newStreamTokenDatabase(t)
	test := newOutboxSyncTestOn(t, database)

	stream := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &stream, "published"))
	test.tasks.Drain()

	// The task claims and stores its Create, and then its transaction fails
	failure := errors.New("the queue lost its connection")
	spool := postcommit.NewTasks()

	_, err := mongodb.NewServer(database).WithTransaction(postcommit.WithContext(t.Context(), spool), func(transaction data.Session) (any, error) {

		if err := test.service.SyncOutbox(transaction, stream.StreamID, OutboxChanges{}); err != nil {
			return nil, err
		}

		// Inside the transaction, the claim and the row are both written
		stored := model.NewStream()
		require.NoError(t, transaction.Collection("Stream").Load(exp.Equal("_id", stream.StreamID), &stored))
		require.NotEmpty(t, stored.Fingerprint.Value())

		return nil, failure
	})

	require.ErrorIs(t, err, failure)
	require.Empty(t, test.fingerprint(t, stream.StreamID), "the claim is taken back")
	require.Zero(t, test.outboxRows(t, &stream, vocab.ActivityTypeCreate), "the outbox row is taken back")

	// The retry sends the Create
	require.NoError(t, test.service.SyncOutbox(test.session, stream.StreamID, OutboxChanges{}))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))
	require.Len(t, test.fingerprint(t, stream.StreamID), 64)
	require.Equal(t, int64(1), test.outboxRows(t, &stream, vocab.ActivityTypeCreate))
}

// TestStreamOutboxPath_Upgrade follows Streams that an older server stored through upgrade 41
// and the fingerprint fill, and confirms what followers are sent afterwards
func TestStreamOutboxPath_Upgrade(t *testing.T) {

	database := newStreamTokenDatabase(t)
	test := newOutboxSyncTestOn(t, database)
	streams := test.session.Collection("Stream")

	// asOlderServer rewrites a stored Stream as a server without the flag wrote it
	asOlderServer := func(streamID primitive.ObjectID, publishDate int64) {
		t.Helper()
		require.NoError(t, queries.RawUpdate(t.Context(), streams, exp.Equal("_id", streamID), bson.M{
			"$unset": bson.M{"isOutbox": "", "fingerprint": ""},
			"$set":   bson.M{"publishDate": publishDate, "unpublishDate": int64(math.MaxInt64)},
		}))
	}

	// A Stream that followers received
	sent := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &sent, "published"))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))
	original := test.fingerprint(t, sent.StreamID)
	asOlderServer(sent.StreamID, sent.PublishDate)

	// A second one, which is deleted after the upgrade
	doomed := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &doomed, "published"))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))
	asOlderServer(doomed.StreamID, doomed.PublishDate)

	// A Stream that was never published, which the older server dated at the end of time
	draft := test.newStream("default")
	require.NoError(t, test.service.Save(test.session, &draft, "draft"))
	asOlderServer(draft.StreamID, math.MaxInt64)

	// The upgrade flags what followers hold, and sends nothing
	require.NoError(t, upgrades.Version41(context.Background(), database))
	require.Empty(t, test.sync(t))

	stored := storedStream(t, test.service, test.session, sent.StreamID)
	require.True(t, stored.IsOutbox.Value())
	require.True(t, stored.IsPublished())
	require.Equal(t, model.StreamFingerprintPending, stored.Fingerprint.Value())
	require.Equal(t, sent.PublishDate, stored.PublishDate)

	storedDraft := storedStream(t, test.service, test.session, draft.StreamID)
	require.False(t, storedDraft.IsOutbox.Value())
	require.Empty(t, storedDraft.Fingerprint.Value())
	require.Zero(t, storedDraft.PublishDate)

	// A Stream deleted before the fill reaches it still sends its Delete
	// Delete reads its Stream in the background, so this copy is not used again
	deleted := storedStream(t, test.service, test.session, doomed.StreamID)
	require.NoError(t, test.service.Delete(test.session, &deleted, "deleted"))
	require.Equal(t, []string{vocab.ActivityTypeDelete}, test.sync(t))

	// The fill records the fingerprint that the older server's Create would have had, and sends nothing
	_, done, err := test.service.FillFingerprints(test.session, primitive.NilObjectID)
	require.NoError(t, err)
	require.True(t, done)
	require.Empty(t, test.sync(t))
	require.Equal(t, original, test.fingerprint(t, sent.StreamID))

	// So a save that changes nothing sends nothing, and an edit sends an Update
	loaded := storedStream(t, test.service, test.session, sent.StreamID)
	require.NoError(t, test.service.Save(test.session, &loaded, "unchanged"))
	require.Empty(t, test.sync(t))

	loaded.Label = "No Fate"
	require.NoError(t, test.service.Save(test.session, &loaded, "edited"))
	require.Equal(t, []string{vocab.ActivityTypeUpdate}, test.sync(t))

	// The draft is published for the first time, and is dated then
	storedDraft.StateID.Set("published")
	require.NoError(t, test.service.Save(test.session, &storedDraft, "published"))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

	published := storedStream(t, test.service, test.session, draft.StreamID)
	require.Positive(t, published.PublishDate)
	require.Less(t, published.PublishDate, int64(math.MaxInt64))
}
