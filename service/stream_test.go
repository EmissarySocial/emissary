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
	refused("signin", "a server route")
	refused("SignIn", "a server route in another case")
	refused("oauth", "a server route")
}

// TestStream_ValidateToken_RefusesEveryReservedToken confirms that each reserved route is refused before
// the database is consulted
func TestStream_ValidateToken_RefusesEveryReservedToken(t *testing.T) {

	service, session := newStreamValidateTestService(model.NewUser())

	for _, token := range ReservedStreamTokens() {
		err := service.ValidateToken(session, primitive.NewObjectID(), token)
		require.Error(t, err, token)
		require.True(t, derp.IsBadRequest(err), token)
		require.Equal(t, "This token is reserved by the server", derp.Message(err), token)
	}

	require.Zero(t, session.streams.loads, "a reserved token must be refused without a lookup")
	require.Zero(t, session.users.loads, "a reserved token must be refused without a lookup")
}

// TestReservedStreamTokens_ReturnsACopy confirms that a caller cannot change the reserved list
func TestReservedStreamTokens_ReturnsACopy(t *testing.T) {

	tokens := ReservedStreamTokens()
	require.NotEmpty(t, tokens)

	tokens[0] = "changed"
	require.NotContains(t, ReservedStreamTokens(), "changed")
	require.False(t, isReservedStreamToken("changed"))
}

// TestStream_ValidateChangedToken pins when Save checks a token: only when it differs from the stored
// copy, and never when it is the Stream's own ID
func TestStream_ValidateChangedToken(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "alice"
	user.IsPublic = true

	// "alice" was saved before tokens and usernames were kept apart, so it now clashes with a username
	shadowed := newActorStream("group", "alice")
	mine := newActorStream("group", "my-article")
	theirs := newActorStream("group", "their-article")

	setup := func() (*Stream, webfingerSession) {
		return newStreamValidateTestService(user, shadowed, mine, theirs)
	}

	t.Run("the default token is never checked", func(t *testing.T) {
		service, session := setup()
		stream := newActorStream("group", "")
		stream.Token = stream.StreamID.Hex()

		require.NoError(t, service.validateChangedToken(session, &stream, true))
		require.NoError(t, service.validateChangedToken(session, &stream, false))
		require.Zero(t, session.streams.loads)
		require.Zero(t, session.users.loads)
	})

	t.Run("an unchanged token is not re-checked", func(t *testing.T) {
		service, session := setup()
		stream := shadowed

		require.NoError(t, service.validateChangedToken(session, &stream, false))
		require.Equal(t, 1, session.streams.loads, "only the stored copy is loaded")
		require.Zero(t, session.users.loads)
	})

	t.Run("a changed token is checked", func(t *testing.T) {
		service, session := setup()
		stream := mine

		stream.Token = "their-article"
		err := service.validateChangedToken(session, &stream, false)
		require.True(t, derp.IsBadRequest(err), "another Stream's token")

		stream.Token = "signin"
		err = service.validateChangedToken(session, &stream, false)
		require.True(t, derp.IsBadRequest(err), "a server route")

		stream.Token = "alice"
		err = service.validateChangedToken(session, &stream, false)
		require.True(t, derp.IsBadRequest(err), "a username, even though another Stream already holds it")

		stream.Token = "something-new"
		require.NoError(t, service.validateChangedToken(session, &stream, false))
	})

	t.Run("a new Stream is checked without loading a stored copy", func(t *testing.T) {
		service, session := setup()
		stream := newActorStream("group", "their-article")

		err := service.validateChangedToken(session, &stream, true)
		require.True(t, derp.IsBadRequest(err))

		stream.Token = "something-new"
		loads := session.streams.loads
		require.NoError(t, service.validateChangedToken(session, &stream, true))
		require.Equal(t, loads+1, session.streams.loads, "only ValidateToken's own lookup by token")
	})

	t.Run("a Stream with no stored copy is checked", func(t *testing.T) {
		service, session := setup()
		stream := newActorStream("group", "their-article")

		err := service.validateChangedToken(session, &stream, false)
		require.True(t, derp.IsBadRequest(err))
	})

	t.Run("a failed load is not a pass", func(t *testing.T) {
		service, session := setup()
		session.streams.loadError = derp.Internal("test", "database unavailable")
		stream := mine

		err := service.validateChangedToken(session, &stream, false)
		require.Error(t, err)
		require.Equal(t, 500, derp.ErrorCode(err))
	})
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
