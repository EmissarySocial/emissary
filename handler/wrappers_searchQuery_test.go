package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/handler/activitypub_search"
	"github.com/EmissarySocial/emissary/model"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// RULE: A SearchQuery actor has no Stream, so none of its handlers may accept one.  This
// assignment is the enforcement: reintroducing a Stream or Template parameter on any of the
// four handlers below breaks the build here rather than in production. (BUG-148)
var _ = []WithFunc1[model.SearchQuery]{
	activitypub_search.GetJSONLD,
	activitypub_search.PostInbox,
	activitypub_search.GetOutboxCollection,
	activitypub_search.GetOutboxMessage,
}

// A route that declares no ":stream" parameter still gets a stream token, and it is the one
// that triggers the startup redirect in WithStream.  This is why WithSearchQuery must not be
// built on WithStream. (BUG-148)
func TestGetStreamToken_AbsentParameterBecomesHome(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "/@search_6a4c2b5179277aea29cce04e", nil)
	ctx := echo.New().NewContext(request, httptest.NewRecorder())

	require.Equal(t, "home", getStreamToken(ctx), "an absent :stream parameter resolves to the home page")
}

// The zero ObjectID is spelled differently and means the same thing.
func TestGetStreamToken_ZeroObjectIDBecomesHome(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := echo.New().NewContext(request, httptest.NewRecorder())
	ctx.SetParamNames("stream")
	ctx.SetParamValues("000000000000000000000000")

	require.Equal(t, "home", getStreamToken(ctx))
}

// A real token is passed through untouched.
func TestGetStreamToken_RealTokenIsPreserved(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "/albums", nil)
	ctx := echo.New().NewContext(request, httptest.NewRecorder())
	ctx.SetParamNames("stream")
	ctx.SetParamValues("albums")

	require.Equal(t, "albums", getStreamToken(ctx))
}
