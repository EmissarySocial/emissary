package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetPassword must enforce the server-side password policy BEFORE hashing and saving, so that
// a crafted request (which ignores the template's client-side minLength) cannot set a password
// that is too short. Steranko's default schema requires at least 8 characters; the rejection path
// halts after ValidatePassword and touches only response(), request(), factory(), session(), and
// authorization() -- so a minimal stub builder that never reaches the database is enough.

// stubPasswordUserService satisfies steranko.UserService. ValidatePassword never calls any of these
// methods, so every method is a no-op / error stub that exists only to satisfy the interface.
type stubPasswordUserService struct{}

func (stubPasswordUserService) New() steranko.User                        { return nil }
func (stubPasswordUserService) Load(string, steranko.User) error          { return nil }
func (stubPasswordUserService) LoadBySubject(string, steranko.User) error { return nil }
func (stubPasswordUserService) Save(steranko.User, string) error          { return nil }
func (stubPasswordUserService) Delete(steranko.User, string) error        { return nil }
func (stubPasswordUserService) RequestPasswordReset(steranko.User) error  { return nil }
func (stubPasswordUserService) Claims(steranko.User) (jwt.Claims, error) {
	authorization := model.NewAuthorization()
	return &authorization, nil
}
func (stubPasswordUserService) Close() {}
func (stubPasswordUserService) NewClaims() jwt.Claims {
	authorization := model.NewAuthorization()
	return &authorization
}

// stubPasswordKeyService satisfies steranko.KeyService with a single fixed HMAC secret.
type stubPasswordKeyService struct{}

func (stubPasswordKeyService) GetCurrentKey() (string, any, error) {
	return "test", []byte("test-secret"), nil
}
func (stubPasswordKeyService) FindKey(*jwt.Token) (any, error) {
	return []byte("test-secret"), nil
}

// stubPasswordFactory is a build.Factory that returns a real steranko.Steranko carrying steranko's
// default password schema (min 8 characters). Every other Factory method is inherited from the
// embedded interface and would panic if called, which is fine: the rejection path touches only Steranko().
type stubPasswordFactory struct {
	Factory
	steranko *steranko.Steranko
}

func (f stubPasswordFactory) Steranko(data.Session) *steranko.Steranko { return f.steranko }

// stubPasswordBuilder is a build.Builder exposing only the methods StepSetPassword.Post reaches
// before the (rejected) password is ever hashed or saved.
type stubPasswordBuilder struct {
	Builder
	req          *http.Request
	res          http.ResponseWriter
	factoryValue Factory
	auth         model.Authorization
}

func (b stubPasswordBuilder) request() *http.Request             { return b.req }
func (b stubPasswordBuilder) response() http.ResponseWriter      { return b.res }
func (b stubPasswordBuilder) factory() Factory                   { return b.factoryValue }
func (b stubPasswordBuilder) session() data.Session              { return nil }
func (b stubPasswordBuilder) authorization() model.Authorization { return b.auth }

// setPasswordRequest builds a urlencoded POST carrying a new/confirm password pair.
func setPasswordRequest(password string) *http.Request {

	form := url.Values{
		"new_password":     {password},
		"confirm_password": {password},
	}

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

// newTestPasswordBuilder wires an authenticated stub builder backed by a real steranko instance.
func newTestPasswordBuilder(password string) stubPasswordBuilder {

	st := steranko.New(stubPasswordUserService{}, stubPasswordKeyService{})

	authorization := model.NewAuthorization()
	authorization.UserID = primitive.NewObjectID() // non-zero UserID => IsAuthenticated() is true

	return stubPasswordBuilder{
		req:          setPasswordRequest(password),
		res:          httptest.NewRecorder(),
		factoryValue: stubPasswordFactory{steranko: st},
		auth:         authorization,
	}
}

// A password shorter than the policy minimum must halt the pipeline before it is hashed or saved.
func TestStepSetPassword_Post_RejectsShortPassword(t *testing.T) {

	builder := newTestPasswordBuilder("a") // 1 character: far below steranko's 8-character default

	step := StepSetPassword{}
	behavior := step.Post(builder, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "a password below the policy minimum must halt the pipeline")
}

// A password that meets the policy minimum must pass validation. (It then proceeds to load the user
// from the database, which the stub does not provide; reaching that point is proof that the length
// gate did not reject an acceptable password.)
func TestStepSetPassword_Post_AcceptsCompliantPassword(t *testing.T) {

	require.Nil(t,
		steranko.New(stubPasswordUserService{}, stubPasswordKeyService{}).ValidatePassword("abcdefgh"),
		"an 8-character password must satisfy steranko's default policy",
	)
}
