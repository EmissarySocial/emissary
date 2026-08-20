package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// StepSetResponse is a thin forwarder to service.Response.SetResponse, which validates the reaction
// target. This test confirms the step surfaces that rejection as a halting error rather than
// silently accepting an invalid target. A malformed URL is rejected by SetResponse's cheap
// syntactic gate BEFORE any object load, so a bare (un-Refreshed) Response service is enough --
// no ActivityStream/network stack, and the unexported loader seam is never reached.

// stubResponseFactory is a build.Factory that returns the provided Response service. Every other
// Factory method is left nil (inherited from the embedded interface) and would panic if called,
// which is fine: StepSetResponse.Post touches only Response().
type stubResponseFactory struct {
	Factory
	responseService *service.Response
}

// Response implements the build.Factory interface, returning this stub's Response service
func (f stubResponseFactory) Response() *service.Response { return f.responseService }

// stubResponseBuilder is a build.Builder that exposes only the four methods StepSetResponse.Post
// uses: request(), getUser(), factory(), and session().
type stubResponseBuilder struct {
	Builder
	req          *http.Request
	user         *model.User
	factoryValue Factory
}

// request implements the Builder interface, returning this stub's request
func (b stubResponseBuilder) request() *http.Request { return b.req }

// getUser implements the Builder interface, returning this stub's signed-in User
func (b stubResponseBuilder) getUser() (*model.User, error) { return b.user, nil }

// factory implements the Builder interface, returning this stub's factory
func (b stubResponseBuilder) factory() Factory { return b.factoryValue }

// session implements the Builder interface. The stub owns no database session.
func (b stubResponseBuilder) session() data.Session { return nil }

// setResponseRequest builds a urlencoded POST that StepSetResponse.Post can bind.
func setResponseRequest(object string) *http.Request {

	form := url.Values{
		"url":    {object},
		"type":   {vocab.ActivityTypeLike},
		"exists": {"true"},
	}

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

// A malformed reaction target must halt the pipeline with an error -- the step cannot let an
// invalid object reach the database.
func TestStepSetResponse_Post_RejectsInvalidTarget(t *testing.T) {

	responseService := service.NewResponse()
	user := model.NewUser()

	builder := stubResponseBuilder{
		req:          setResponseRequest("not-a-url"),
		user:         &user,
		factoryValue: stubResponseFactory{responseService: &responseService},
	}

	behavior := StepSetResponse{}.Post(builder, io.Discard)

	// Apply the returned behavior to a fresh result and inspect it.
	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "an invalid reaction target must halt the pipeline")
	require.NotNil(t, result.Error, "an invalid reaction target must produce an error")
}
