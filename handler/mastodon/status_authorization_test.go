package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests lock in the WRITE-side authorization for the Mastodon status handlers
// (PutStatus, DeleteStatus). Writes are author-only, matching the Mastodon API model
// where a status belongs to a single account. This is deliberately independent of the
// template's access list (that governs reads, via userCanStream / UserCan).
//
// The regression being guarded against is twofold: (1) the original bug used
// Stream.IsMyself, which always returns false and rejected even the real author;
// (2) routing writes through the template's "edit"/"delete" role could grant a
// non-author write access if a template shared those roles with a group.

// ownedStream builds a Stream authored by authorID.
func ownedStream(authorID primitive.ObjectID) model.Stream {
	stream := model.NewStream()
	stream.AttributedTo = model.PersonLink{UserID: authorID}
	return stream
}

// TestUserOwnsStream_Author_Allowed verifies that a Stream's author may act on it
func TestUserOwnsStream_Author_Allowed(t *testing.T) {

	author := primitive.NewObjectID()
	stream := ownedStream(author)

	auth := model.NewAuthorization()
	auth.UserID = author

	require.Nil(t, userOwnsStream(&auth, &stream), "the author must be allowed to modify their own stream")
}

// TestUserOwnsStream_OtherUser_Denied verifies that a different signed-in User may not act on a Stream
func TestUserOwnsStream_OtherUser_Denied(t *testing.T) {

	author := primitive.NewObjectID()
	stream := ownedStream(author)

	auth := model.NewAuthorization()
	auth.UserID = primitive.NewObjectID()

	err := userOwnsStream(&auth, &stream)
	require.NotNil(t, err, "a non-author user must NOT be allowed to modify the stream")
	require.True(t, derp.IsForbidden(err), "the denial must be a Forbidden error")
}

// TestUserOwnsStream_DomainOwner_Allowed verifies that a domain owner may act on any Stream
func TestUserOwnsStream_DomainOwner_Allowed(t *testing.T) {

	author := primitive.NewObjectID()
	stream := ownedStream(author)

	// A domain owner (not the author) may modify the stream, so server moderation
	// can take a post down.
	auth := model.NewAuthorization()
	auth.UserID = primitive.NewObjectID()
	auth.DomainOwner = true

	require.Nil(t, userOwnsStream(&auth, &stream), "a domain owner must be allowed to modify any stream")
}

// TestUserOwnsStream_Anonymous_Denied verifies that an anonymous request may not act on a Stream
func TestUserOwnsStream_Anonymous_Denied(t *testing.T) {

	author := primitive.NewObjectID()
	stream := ownedStream(author)

	// An anonymous caller (zero UserID) must be denied.
	auth := model.NewAuthorization()

	err := userOwnsStream(&auth, &stream)
	require.NotNil(t, err, "an anonymous caller must NOT be allowed to modify the stream")
	require.True(t, derp.IsForbidden(err), "the denial must be a Forbidden error")
}

// TestUserOwnsStream_AnonymousAgainstZeroAuthor_Denied verifies that a zero UserID does not match an unauthored Stream
func TestUserOwnsStream_AnonymousAgainstZeroAuthor_Denied(t *testing.T) {

	// Edge case: a Stream with no author (zero AttributedTo.UserID) must not be
	// writable by an anonymous caller (zero UserID). IsAuthor guards against a
	// zero authorID, so this must still deny rather than match zero-to-zero.
	stream := model.NewStream()

	auth := model.NewAuthorization()

	err := userOwnsStream(&auth, &stream)
	require.NotNil(t, err, "a zero-UserID caller must NOT match a zero-author stream")
	require.True(t, derp.IsForbidden(err), "the denial must be a Forbidden error")
}
