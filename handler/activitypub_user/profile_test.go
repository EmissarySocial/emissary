package activitypub_user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests cover the visibility gate that hides a non-public actor document (isPublic=false,
// "Hidden from Public Servers") from anonymous/remote requesters, while still letting the domain
// owner and the user themselves fetch their own actor JSON -- matching the HTML path and every
// sibling ActivityPub collection (outbox/following/featured/blocked/key).

/******************************************
 * Steranko test harness
 *
 * A minimal Steranko lets us mint real (HMAC-signed) authorizations so isUserVisible reads them
 * back through the production code path. Anonymous requests carry no token.
 ******************************************/

// testJWTSecret is the fixed HMAC secret that the stub key service signs and verifies with
const testJWTSecret = "profile-test-secret-key"

// stubKeyService signs and verifies with a single fixed HMAC secret.
type stubKeyService struct{}

// GetCurrentKey implements the steranko KeyService interface, returning the fixed test secret
func (stubKeyService) GetCurrentKey() (string, any, error) {
	return "test", []byte(testJWTSecret), nil
}

// FindKey implements the steranko KeyService interface, returning the fixed test secret
func (stubKeyService) FindKey(*jwt.Token) (any, error) {
	return []byte(testJWTSecret), nil
}

// stubUserService supplies only what the Authorization-parsing path needs: NewClaims must return
// a *model.Authorization so the parsed token populates it. Everything else is an unused no-op.
type stubUserService struct{}

// New implements the steranko UserService interface. Unused by these tests.
func (stubUserService) New() steranko.User { return nil }

// Load implements the steranko UserService interface. Unused by these tests.
func (stubUserService) Load(string, steranko.User) error { return derp.NotFound("test", "unused") }

// LoadBySubject implements the steranko UserService interface. Unused by these tests.
func (stubUserService) LoadBySubject(string, steranko.User) error {
	return derp.NotFound("test", "unused")
}

// Save implements the steranko UserService interface. Unused by these tests.
func (stubUserService) Save(steranko.User, string) error { return derp.Internal("test", "unused") }

// Delete implements the steranko UserService interface. Unused by these tests.
func (stubUserService) Delete(steranko.User, string) error { return derp.Internal("test", "unused") }

// RequestPasswordReset implements the steranko UserService interface. Unused by these tests.
func (stubUserService) RequestPasswordReset(steranko.User) error {
	return derp.Internal("test", "unused")
}

// Claims implements the steranko UserService interface. Unused by these tests.
func (stubUserService) Claims(steranko.User) (jwt.Claims, error) {
	return nil, derp.Internal("test", "unused")
}

// Close implements the steranko UserService interface. The stub holds no resources to release.
func (stubUserService) Close() {}

// NewClaims implements the steranko UserService interface, returning an empty Authorization to parse into
func (stubUserService) NewClaims() jwt.Claims {
	authorization := model.NewAuthorization()
	return &authorization
}

// newTestContext wraps a request in a steranko.Context backed by the stub services. When an
// authorization is supplied, it is signed and attached as a Bearer token.
func newTestContext(t *testing.T, authorization *model.Authorization) *steranko.Context {
	t.Helper()

	st := steranko.New(stubUserService{}, stubKeyService{})

	request := httptest.NewRequest(http.MethodGet, "/@qapwd", nil)

	if authorization != nil {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, authorization)
		signed, err := token.SignedString([]byte(testJWTSecret))
		require.Nil(t, err)
		request.Header.Set("Authorization", "Bearer "+signed)
	}

	recorder := httptest.NewRecorder()
	return st.Context(echo.New().NewContext(request, recorder))
}

/******************************************
 * isUserVisible tests
 ******************************************/

// TestIsUserVisible_Anonymous_Public confirms that a public profile is visible to anonymous requesters.
func TestIsUserVisible_Anonymous_Public(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.IsPublic = true

	require.True(t, isUserVisible(newTestContext(t, nil), &user))
}

// TestIsUserVisible_Anonymous_NonPublic is the core of the reported bug: a non-public profile
// must be hidden from anonymous requesters.
func TestIsUserVisible_Anonymous_NonPublic(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.IsPublic = false

	require.False(t, isUserVisible(newTestContext(t, nil), &user))
}

// TestIsUserVisible_Self confirms that a signed-in user can always see their own profile,
// even when it is non-public.
func TestIsUserVisible_Self(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.IsPublic = false

	authorization := model.NewAuthorization()
	authorization.UserID = user.UserID

	require.True(t, isUserVisible(newTestContext(t, &authorization), &user))
}

// TestIsUserVisible_DomainOwner confirms that a domain owner can see any profile, even non-public.
func TestIsUserVisible_DomainOwner(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.IsPublic = false

	authorization := model.NewAuthorization()
	authorization.UserID = primitive.NewObjectID() // a different user
	authorization.DomainOwner = true

	require.True(t, isUserVisible(newTestContext(t, &authorization), &user))
}

// TestIsUserVisible_OtherUser confirms that a signed-in user who is NOT the owner and NOT the
// profile's owner is still blocked from a non-public profile.
func TestIsUserVisible_OtherUser(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.IsPublic = false

	authorization := model.NewAuthorization()
	authorization.UserID = primitive.NewObjectID() // a different, non-owner user

	require.False(t, isUserVisible(newTestContext(t, &authorization), &user))
}

/******************************************
 * RenderProfileJSONLD tests
 ******************************************/

// TestRenderProfileJSONLD_Anonymous_NonPublic is the regression test for the reported content-
// negotiation leak: an anonymous ActivityPub request for a non-public actor document must be
// rejected with NotFound before the actor JSON (name, key, followers/following URLs) is assembled.
// The gate runs before any factory/session access, so both can be nil here.
func TestRenderProfileJSONLD_Anonymous_NonPublic(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "qapwd"
	user.IsPublic = false

	err := RenderProfileJSONLD(newTestContext(t, nil), nil, nil, &user)

	require.NotNil(t, err)
	require.True(t, derp.IsNotFound(err))
}
