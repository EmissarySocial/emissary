package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newAudienceStream returns a published Stream by authorID that mentions one remote actor, with the
// DefaultAllow that the provided Groups or Circles produce for its viewers
func newAudienceStream(authorID primitive.ObjectID, defaultAllow model.Permissions, circles id.Slice) model.Stream {

	stream := newTagStream("Hey @sarah@mastodon.social")
	stream.StreamID = primitive.NewObjectID()
	stream.ParentID = primitive.NewObjectID()
	stream.AttributedTo.UserID = authorID
	stream.DefaultAllow = defaultAllow
	stream.Circles["viewer"] = circles
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "sarah@mastodon.social",
		Href: "https://mastodon.social/users/sarah",
	}}

	return stream
}

// newAudienceService returns a Stream service that can render JSON-LD and build addressing on https://example.com
func newAudienceService() *Stream {
	streamService := newMentionStreamService()
	streamService.locatorService = &Locator{host: "https://example.com"}
	return streamService
}

// TestStream_JSONLD_Audience verifies that a published object names everyone its author sends it to,
// because delivery reads addressing alone
func TestStream_JSONLD_Audience(t *testing.T) {

	streamService := newAudienceService()
	authorID := primitive.NewObjectID()
	circleID := primitive.NewObjectID()

	authorFollowers := "https://example.com/@" + authorID.Hex() + "/pub/followers"
	circleURL := "https://example.com/@" + authorID.Hex() + "/pub/circles/" + circleID.Hex()
	mention := "https://mastodon.social/users/sarah"

	t.Run("public", func(t *testing.T) {
		stream := newAudienceStream(authorID, model.NewAnonymousPermissions(), nil)
		document := streamService.JSONLD(emptyTagSession{}, &stream)

		require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, document.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, []string{authorFollowers, mention}, document.GetSliceOfString(vocab.PropertyCC))
	})

	t.Run("circle-only", func(t *testing.T) {
		stream := newAudienceStream(authorID, model.Permissions{circleID}, id.Slice{circleID})
		document := streamService.JSONLD(emptyTagSession{}, &stream)

		require.Equal(t, []string{circleURL}, document.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, []string{mention}, document.GetSliceOfString(vocab.PropertyCC), "a follower outside the circle is never named")
	})

	t.Run("signed-in visitors", func(t *testing.T) {
		stream := newAudienceStream(authorID, model.NewAuthenticatedPermissions(), nil)
		document := streamService.JSONLD(emptyTagSession{}, &stream)

		require.NotContains(t, document, vocab.PropertyTo)
		require.Equal(t, []string{mention}, document.GetSliceOfString(vocab.PropertyCC))
	})

	t.Run("a mention of the author's own followers is not repeated", func(t *testing.T) {
		stream := newAudienceStream(authorID, model.NewAnonymousPermissions(), nil)
		stream.Tags = model.TagList{{Type: vocab.LinkTypeMention, Name: "followers", Href: authorFollowers}}
		document := streamService.JSONLD(emptyTagSession{}, &stream)

		require.Equal(t, []string{authorFollowers}, document.GetSliceOfString(vocab.PropertyCC))
	})
}

// TestStream_AudienceMap verifies that the audience a Delete falls back to is the one the object names
func TestStream_AudienceMap(t *testing.T) {

	streamService := newAudienceService()
	circleID := primitive.NewObjectID()
	stream := newAudienceStream(primitive.NewObjectID(), model.Permissions{circleID}, id.Slice{circleID})

	document := streamService.JSONLD(emptyTagSession{}, &stream)
	audience := streamService.audienceMap(&stream)

	require.Equal(t, document.GetSliceOfString(vocab.PropertyTo), audience.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, document.GetSliceOfString(vocab.PropertyCC), audience.GetSliceOfString(vocab.PropertyCC))
}

// TestStream_ParentAudience verifies that a parent Stream announces a public child to Public and its own
// followers, and a circle-only child to no one
func TestStream_ParentAudience(t *testing.T) {

	streamService := newAudienceService()
	circleID := primitive.NewObjectID()

	public := newAudienceStream(primitive.NewObjectID(), model.NewAnonymousPermissions(), nil)
	audience := streamService.parentAudience(&public)
	require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, audience.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, []string{"https://example.com/" + public.ParentID.Hex() + "/pub/followers"}, audience.GetSliceOfString(vocab.PropertyCC))

	circleOnly := newAudienceStream(primitive.NewObjectID(), model.Permissions{circleID}, id.Slice{circleID})
	audience = streamService.parentAudience(&circleOnly)
	require.Empty(t, audience.GetSliceOfString(vocab.PropertyTo))
	require.Empty(t, audience.GetSliceOfString(vocab.PropertyCC))
}
