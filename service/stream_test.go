package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStreamService verifies that the Stream service satisfies the ModelService interface
func TestStreamService(t *testing.T) {
	var service any = &Stream{}
	_, ok := service.(ModelService)
	require.True(t, ok)
}

// newStreamValidateTestService returns a Stream service wired to a User service, both backed by the
// in-memory session from stream_webfinger_test.go, holding the provided Streams and one public User.
func newStreamValidateTestService(user model.User, streams ...model.Stream) (*Stream, webfingerSession) {

	service, session := newStreamWebFingerService([]model.Template{actorTemplate("group")}, streams...)
	service.userService = &User{host: "https://example.com"}

	session.users.record = user
	session.users.found = true

	return service, session
}

// TestStream_ValidateToken pins the rules for assigning a token (BUG-98): long enough, not another
// Stream's token or id, and not a username in any letter case, which would shadow the Stream in WebFinger.
func TestStream_ValidateToken(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "alice"
	user.IsPublic = true

	mine := newActorStream("group", "my-article")
	theirs := newActorStream("group", "their-article")
	service, session := newStreamValidateTestService(user, mine, theirs)

	refused := func(token string, reason string) {
		err := service.ValidateToken(session, mine.StreamID, token)
		require.Error(t, err, reason)
		require.True(t, derp.IsBadRequest(err), reason)
	}

	// Available tokens
	require.NoError(t, service.ValidateToken(session, mine.StreamID, "my-article"), "a Stream may keep its own token")
	require.NoError(t, service.ValidateToken(session, mine.StreamID, mine.StreamID.Hex()), "a Stream may use its own id")
	require.NoError(t, service.ValidateToken(session, mine.StreamID, "something-new"), "an unused token is available")
	require.NoError(t, service.ValidateToken(session, primitive.NilObjectID, "something-new"), "a new Stream may take an unused token")

	// Refused tokens
	refused("ab", "too short")
	refused("their-article", "another Stream's token")
	refused(theirs.StreamID.Hex(), "another Stream's id")
	refused("alice", "a username")
	refused("Alice", "a username in another case")
	refused("ALICE", "a username in upper case")
}

// TestStream_ValidateToken_ErrorsAreNotAvailability confirms that a database failure on either
// lookup is reported, rather than reading as an available token.
func TestStream_ValidateToken_ErrorsAreNotAvailability(t *testing.T) {

	service, session := newStreamValidateTestService(model.NewUser())

	session.streams.loadError = derp.Internal("test", "database unavailable")
	err := service.ValidateToken(session, primitive.NilObjectID, "something-new")
	require.Error(t, err)
	require.Equal(t, 500, derp.ErrorCode(err))

	session.streams.loadError = nil
	session.users.loadError = derp.Internal("test", "database unavailable")
	err = service.ValidateToken(session, primitive.NilObjectID, "something-new")
	require.Error(t, err)
	require.Equal(t, 500, derp.ErrorCode(err))
}
