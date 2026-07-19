package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// The Like/Dislike intent POST handlers forward the caller-supplied `object` to
// service.Response.SetResponse, which validates the reaction target. These tests confirm the
// handlers surface that rejection (they do NOT store a Response for a bogus target) rather than
// returning 200.
//
// A malformed URL is rejected by SetResponse's cheap syntactic gate BEFORE any object load or DB
// read, so a zero Factory + a bare Response service is enough: the network/Mongo stack is never
// reached. (The unresolvable-but-well-formed case needs the ActivityStream loader, which is only
// wired up by Factory.Refresh against a live domain, so it is covered by the service-level tests.)

// intentRequest builds a urlencoded POST carrying the reaction target.
func intentRequest(object string) *steranko.Context {

	form := url.Values{"object": {object}}

	request := httptest.NewRequest(http.MethodPost, "/@me/intent/like", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	return &steranko.Context{Context: echo.New().NewContext(request, recorder)}
}

// A Like intent on a malformed target must return an error, not a 200 with a stored Response.
func TestPostIntent_Like_RejectsInvalidTarget(t *testing.T) {

	ctx := intentRequest("not-a-url")
	factory := &service.Factory{}
	user := model.NewUser()

	err := PostIntent_Like(ctx, factory, nil, &user)

	require.NotNil(t, err, "an invalid reaction target must be rejected")
}

// The Dislike intent shares postIntent_Response, so it must reject the same way.
func TestPostIntent_Dislike_RejectsInvalidTarget(t *testing.T) {

	ctx := intentRequest("not-a-url")
	factory := &service.Factory{}
	user := model.NewUser()

	err := PostIntent_Dislike(ctx, factory, nil, &user)

	require.NotNil(t, err, "an invalid reaction target must be rejected")
}
