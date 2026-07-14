package service

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/stretchr/testify/require"
)

// TestSterankoUserServiceInterface confirms that SterankoUserService satisfies
// the steranko.UserService interface, including the LoadBySubject method added
// for session revalidation. This is a compile-time guarantee made explicit.
func TestSterankoUserServiceInterface(t *testing.T) {
	var service any = SterankoUserService{}
	_, ok := service.(steranko.UserService)
	require.True(t, ok)
}

// TestSterankoUserService_LoadBySubject_InvalidType confirms that LoadBySubject
// rejects a non-User pointer before doing any work.
func TestSterankoUserService_LoadBySubject_InvalidType(t *testing.T) {
	service := SterankoUserService{}

	// A nil/non-*model.User result is a programming error and must be reported.
	err := service.LoadBySubject("000000000000000000000000", nil)
	require.NotNil(t, err)
	require.True(t, derp.IsInternalServerError(err))
}

// TestSterankoUserService_LoadBySubject_InvalidSubject confirms that a subject
// that is not a hex ObjectID is rejected before any database access. The "sub"
// claim must be the User's hex-encoded UserID (see claims()).
func TestSterankoUserService_LoadBySubject_InvalidSubject(t *testing.T) {
	service := SterankoUserService{}

	user := service.New() // a fresh *model.User

	err := service.LoadBySubject("not-a-hex-object-id", user)
	require.NotNil(t, err)
}
