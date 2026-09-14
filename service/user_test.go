package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newUserValidateTestService returns a User service wired to a Stream service, both backed by the
// in-memory session from stream_webfinger_test.go. The User collection starts empty.
func newUserValidateTestService(streams ...model.Stream) (*User, webfingerSession) {

	streamService, session := newStreamWebFingerService([]model.Template{actorTemplate("group")}, streams...)
	service := &User{host: "https://example.com", streamService: streamService}

	return service, session
}

// TestUser_ValidateUsername_RefusesStreamToken pins the second half of the acct: namespace rule
// (BUG-98): a username that matches an existing Stream token, in any letter case, is refused,
// because WebFinger would then resolve that Stream's handle to the User instead.
func TestUser_ValidateUsername_RefusesStreamToken(t *testing.T) {

	service, session := newUserValidateTestService(newActorStream("group", "alice"))

	for _, username := range []string{"alice", "Alice", "ALICE"} {
		err := service.ValidateUsername(session, primitive.NewObjectID(), username)
		require.Error(t, err, "username %q", username)
		require.True(t, derp.IsBadRequest(err), "username %q", username)
	}
}

// TestUser_ValidateUsername_FreeUsername confirms that a username no Stream token claims is accepted
func TestUser_ValidateUsername_FreeUsername(t *testing.T) {

	service, session := newUserValidateTestService(newActorStream("group", "alice"))

	require.NoError(t, service.ValidateUsername(session, primitive.NewObjectID(), "bob"))
}

// TestUser_ValidateUsername_StreamErrorIsNotAvailability confirms that a database failure while
// checking Stream tokens is reported, rather than reading as an available username.
func TestUser_ValidateUsername_StreamErrorIsNotAvailability(t *testing.T) {

	service, session := newUserValidateTestService()
	session.streams.loadError = derp.Internal("test", "database unavailable")

	err := service.ValidateUsername(session, primitive.NewObjectID(), "bob")

	require.Error(t, err)
	require.Equal(t, 500, derp.ErrorCode(err))
}
