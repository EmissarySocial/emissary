package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	mockdb "github.com/benpate/data-mock"
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

// TestUser_ValidateUsername_RefusesStreamToken pins the other half of the acct: namespace rule (BUG-98):
// a username matching a Stream token in any letter case is refused, or WebFinger would hand that handle to the User.
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

// TestUser_Save_ErrorsCarryNoUser verifies that a refused save names the User by ID, and never carries
// the password hash or a live reset code into the error log
func TestUser_Save_ErrorsCarryNoUser(t *testing.T) {

	user := model.NewUser()
	user.Password = "$2a$12$abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRS"
	user.PasswordReset.AuthCode = "reset-code-that-takes-over-the-account"

	// A missing email address is refused before the service reads anything
	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	err = (&User{}).Save(session, &user, "test")

	require.True(t, derp.IsBadRequest(err))
	secretcheck.RequireAbsent(t, err, user.Password)
	secretcheck.RequireAbsent(t, err, user.PasswordReset.AuthCode)

	found, findErr := secretcheck.Find(err, user.UserID.Hex())
	require.NoError(t, findErr)
	require.NotEmpty(t, found, "the error should still name the User")
}
