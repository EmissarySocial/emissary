package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// mastodonListTest holds a Stream service and the rows that the Mastodon API's lists are tested on
type mastodonListTest struct {
	service *Stream
	session data.Session
	ownerID primitive.ObjectID
}

// newMastodonListTest returns a Stream service on a throwaway database. It skips when no database is reachable.
func newMastodonListTest(t *testing.T) mastodonListTest {

	t.Helper()

	database := newStreamTokenDatabase(t)
	service := newStreamTokenService()

	permissionService := NewPermission()
	service.permissionService = &permissionService

	return mastodonListTest{
		service: service,
		session: newStreamTokenSession(t, database),
		ownerID: primitive.NewObjectID(),
	}
}

// add stores one Stream in the outbox, attributed to the test's owner, after the edit is applied
func (test mastodonListTest) add(t *testing.T, label string, edit func(*model.Stream)) {

	t.Helper()

	stream := model.NewStream()
	stream.TemplateID = "outbox-message"
	stream.ParentID = test.ownerID
	stream.Label = label
	stream.AttributedTo.UserID = test.ownerID
	stream.DefaultAllow = model.NewAnonymousPermissions()
	stream.IsOutbox.Set(true)
	stream.PublishDate = 100

	edit(&stream)

	require.NoError(t, test.session.Collection("Stream").Save(&stream, "test"))
}

// streamLabels returns the label of every Stream
func streamLabels(streams []model.Stream) []string {

	result := make([]string, 0, len(streams))

	for _, stream := range streams {
		result = append(result, stream.Label)
	}

	return result
}

// TestStream_MastodonLists_ShowOnlyTheOutbox stores Streams in and out of the outbox, and
// confirms that every list leaves out the ones that followers were never sent
func TestStream_MastodonLists_ShowOnlyTheOutbox(t *testing.T) {

	test := newMastodonListTest(t)
	const parentURL = "https://remote.example/notes/1"

	hashtag := func(stream *model.Stream) {
		stream.Tags = append(stream.Tags, model.Tag{Type: vocab.LinkTypeHashtag, Name: "emissary"})
	}

	// In the outbox, and open to anyone
	test.add(t, "note", hashtag)

	// A draft: its window is open, as a Stream's was before the upgrade, but it is not in the outbox
	test.add(t, "draft", func(stream *model.Stream) {
		hashtag(stream)
		stream.IsOutbox.Set(false)
		stream.InReplyTo = parentURL
	})

	// In the outbox, and shared with nobody
	test.add(t, "private", func(stream *model.Stream) {
		hashtag(stream)
		stream.DefaultAllow = model.NewPermissions()
	})

	// A reply in the outbox
	test.add(t, "reply", func(stream *model.Stream) {
		hashtag(stream)
		stream.TemplateID = "outbox-reply"
		stream.InReplyTo = parentURL
	})

	// An album in the outbox, stored under a folder and not under the User
	test.add(t, "album", func(stream *model.Stream) {
		hashtag(stream)
		stream.TemplateID = "bandwagon-album"
		stream.ParentID = primitive.NewObjectID()
		stream.InReplyTo = parentURL
	})

	// An article, which is on the site and not in the outbox
	test.add(t, "article", func(stream *model.Stream) {
		stream.TemplateID = "article-markdown"
		stream.SocialRole = vocab.ObjectTypeArticle
		stream.ParentID = primitive.NewObjectID()
		stream.StateID.Set("published")
		stream.IsOutbox.Set(false)
	})

	// In the outbox, and attributed to someone else
	test.add(t, "stranger", func(stream *model.Stream) {
		hashtag(stream)
		stream.ParentID = primitive.NewObjectID()
		stream.AttributedTo.UserID = primitive.NewObjectID()
	})

	anonymous := model.NewAuthorization()

	owner := model.NewAuthorization()
	owner.UserID = test.ownerID

	domainOwner := model.NewAuthorization()
	domainOwner.UserID = primitive.NewObjectID()
	domainOwner.DomainOwner = true

	t.Run("a profile lists everything the User sent", func(t *testing.T) {

		streams, err := test.service.QueryByUser(test.session, anonymous, test.ownerID, exp.All())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"note", "reply", "album"}, streamLabels(streams))
	})

	t.Run("the owner sees their private posts, and still no draft", func(t *testing.T) {

		streams, err := test.service.QueryByUser(test.session, owner, test.ownerID, exp.All())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"note", "private", "reply", "album"}, streamLabels(streams))
	})

	t.Run("the domain owner sees private posts, and still no draft", func(t *testing.T) {

		streams, err := test.service.QueryByUser(test.session, domainOwner, test.ownerID, exp.All())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"note", "private", "reply", "album"}, streamLabels(streams))
	})

	t.Run("the count matches the list", func(t *testing.T) {

		count, newest, err := test.service.SummarizeByUser(test.session, anonymous, test.ownerID)
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
		require.Equal(t, int64(100), newest)
	})

	t.Run("the public timeline holds notes alone", func(t *testing.T) {

		streams, err := test.service.QueryPublic(test.session, owner, exp.All())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"note", "stranger"}, streamLabels(streams))
	})

	t.Run("a hashtag holds notes and replies", func(t *testing.T) {

		streams, err := test.service.QueryByHashtag(test.session, owner, []string{"emissary"}, exp.All())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"note", "reply", "stranger"}, streamLabels(streams))
	})

	t.Run("replies hold notes and replies", func(t *testing.T) {

		streams, err := test.service.QueryReplies(test.session, owner, parentURL)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"reply"}, streamLabels(streams))
	})
}
