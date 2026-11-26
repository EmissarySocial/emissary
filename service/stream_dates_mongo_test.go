package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestStream_Publish_DatesOnlyTheOutbox confirms that Publish gives a date to a Stream that enters
// the outbox, and none to a page that is on the site and not sent to followers
func TestStream_Publish_DatesOnlyTheOutbox(t *testing.T) {

	test := newOutboxSyncTest(t)

	// "hidden" is a State outside the outbox, as an article's "published" State is
	page := newOutboxStream("default")
	require.NoError(t, test.service.Publish(test.session, &test.author, &page, "hidden", false))

	stored := storedStream(t, test.service, test.session, page.StreamID)
	require.False(t, stored.IsOutbox.Value())
	require.Zero(t, stored.PublishDate)
	require.Equal(t, test.author.UserID, stored.AttributedTo.UserID)

	// A State in the outbox dates the Stream
	note := newOutboxStream("default")
	require.NoError(t, test.service.Publish(test.session, &test.author, &note, "published", false))

	stored = storedStream(t, test.service, test.session, note.StreamID)
	require.True(t, stored.IsOutbox.Value())
	require.Positive(t, stored.PublishDate)
}

// TestStream_UnPublish_KeepsTheDate confirms that leaving the outbox clears the flag and leaves
// the publish date alone, so a Stream that comes back keeps its place
func TestStream_UnPublish_KeepsTheDate(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := newOutboxStream("default")
	stream.PublishDate = 1000
	require.NoError(t, test.service.Publish(test.session, &test.author, &stream, "published", false))
	require.NoError(t, test.service.UnPublish(test.session, &stream, "default"))

	stored := storedStream(t, test.service, test.session, stream.StreamID)
	require.False(t, stored.IsOutbox.Value())
	require.False(t, stored.IsPublished())
	require.Equal(t, int64(1000), stored.PublishDate)
}

// TestStream_Move_LeavesTheOutbox confirms that a moved Stream is out of the outbox with no
// fingerprint, and that the move queues no task, so its followers are sent no Delete
func TestStream_Move_LeavesTheOutbox(t *testing.T) {

	test := newOutboxSyncTest(t)
	test.service.notificationService = &Notification{}

	stream := newOutboxStream("default")
	require.NoError(t, test.service.Publish(test.session, &test.author, &stream, "published", false))
	test.sync(t)

	loaded := storedStream(t, test.service, test.session, stream.StreamID)
	require.True(t, loaded.IsOutbox.Value())
	require.NotEmpty(t, loaded.Fingerprint.Value())
	publishDate := loaded.PublishDate
	test.tasks.Drain()

	require.NoError(t, test.service.Move(test.session, &loaded, "https://elsewhere.example/moved"))

	stored := storedStream(t, test.service, test.session, stream.StreamID)
	require.Equal(t, "https://elsewhere.example/moved", stored.MovedTo)
	require.False(t, stored.IsOutbox.Value())
	require.False(t, stored.IsPublished())
	require.Empty(t, stored.Fingerprint.Value())
	require.Equal(t, publishDate, stored.PublishDate, "a moved Stream keeps its date")
	require.Empty(t, test.tasks.Drain())
}

// TestNewStream_HasNoDate confirms that a new Stream holds no date, and no sentinel in place of one
func TestNewStream_HasNoDate(t *testing.T) {
	stream := model.NewStream()
	require.Zero(t, stream.PublishDate)
}
