package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestIsAuthor_Stream covers Stream.IsAuthor, which gates author-level permissions in
// service/permission.go and build/builder_stream.go. The author is the AttributedTo.UserID, and a
// ZERO query ID (an anonymous/unauthenticated request) must never match.
func TestIsAuthor_Stream(t *testing.T) {

	author := primitive.NewObjectID()

	stream := NewStream()
	stream.AttributedTo.UserID = author

	require.True(t, stream.IsAuthor(author))                   // the real author matches
	require.False(t, stream.IsAuthor(primitive.NewObjectID())) // a different user does not
	require.False(t, stream.IsAuthor(primitive.NilObjectID))   // RULE: a zero (anonymous) ID never matches

	// RULE: even when the stream has no author set, a zero query ID must not match a zero author.
	empty := NewStream()
	require.False(t, empty.IsAuthor(primitive.NilObjectID))
}

// TestIsAuthor_Collection covers Collection.IsAuthor (author is UserID, zero never matches).
func TestIsAuthor_Collection(t *testing.T) {

	author := primitive.NewObjectID()

	collection := NewCollection()
	collection.UserID = author

	require.True(t, collection.IsAuthor(author))
	require.False(t, collection.IsAuthor(primitive.NewObjectID()))
	require.False(t, collection.IsAuthor(primitive.NilObjectID))

	empty := NewCollection()
	require.False(t, empty.IsAuthor(primitive.NilObjectID))
}

// TestIsAuthor_Annotation covers Annotation.IsAuthor (author is UserID, zero never matches).
func TestIsAuthor_Annotation(t *testing.T) {

	author := primitive.NewObjectID()

	annotation := NewAnnotation()
	annotation.UserID = author

	require.True(t, annotation.IsAuthor(author))
	require.False(t, annotation.IsAuthor(primitive.NewObjectID()))
	require.False(t, annotation.IsAuthor(primitive.NilObjectID))

	empty := NewAnnotation()
	require.False(t, empty.IsAuthor(primitive.NilObjectID))
}

// TestIsAuthor_KeyPackage covers KeyPackage.IsAuthor (author is UserID, zero never matches).
// The zero-ID guard was added to match the other AccessLister implementations: without it, a
// KeyPackage with a zero UserID would treat an anonymous (zero-ID) request as the author.
func TestIsAuthor_KeyPackage(t *testing.T) {

	author := primitive.NewObjectID()

	keyPackage := NewKeyPackage()
	keyPackage.UserID = author

	require.True(t, keyPackage.IsAuthor(author))
	require.False(t, keyPackage.IsAuthor(primitive.NewObjectID()))
	require.False(t, keyPackage.IsAuthor(primitive.NilObjectID))

	// RULE: a zero-UserID KeyPackage must not match a zero (anonymous) query ID.
	empty := NewKeyPackage()
	require.False(t, empty.IsAuthor(primitive.NilObjectID))
}

// TestIsAuthor_StubsReturnFalse spot-checks that the models which do NOT implement authorship
// (the AccessLister stubs) always report false, regardless of the ID passed.
func TestIsAuthor_StubsReturnFalse(t *testing.T) {

	id := primitive.NewObjectID()

	circle := NewCircle()
	require.False(t, circle.IsAuthor(id))

	folder := NewFolder()
	require.False(t, folder.IsAuthor(id))

	domain := NewDomain()
	require.False(t, domain.IsAuthor(id))
}
