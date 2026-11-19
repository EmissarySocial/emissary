package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestErrorHandler_APIErrorsAreMastodonJSON confirms an API error reaches the client as {"error": ...}
// with the original message, which is the only shape Mastodon apps show, and hides server-failure detail
func TestErrorHandler_APIErrorsAreMastodonJSON(t *testing.T) {

	// send runs one error through the handler, as if raised on an API path of a public host
	send := func(err error) *httptest.ResponseRecorder {
		e := echo.New()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "https://social.example/api/v1/statuses", nil)
		errorHandler(err, e.NewContext(request, recorder))
		return recorder
	}

	// A refusal, wrapped twice on its way out (handler, then toot-echo)
	refused := derp.Wrap(
		derp.Wrap(derp.BadRequest("handler", "Polls are not supported on this server yet", derp.WithCode(http.StatusUnprocessableEntity)), "toot-echo", "Error executing API call"),
		"outer", "Something")

	recorder := send(refused)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Contains(t, recorder.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)
	require.JSONEq(t, `{"error":"Polls are not supported on this server yet"}`, recorder.Body.String())

	// A server failure keeps its detail out of the response
	recorder = send(derp.Wrap(derp.Internal("database", "server selection timeout"), "toot-echo", "Error executing API call"))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"Internal Server Error"}`, recorder.Body.String())
}

// TestErrorHandler_WritesOneBody confirms an error handled twice still sends a single body
func TestErrorHandler_WritesOneBody(t *testing.T) {

	e := echo.New()
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "https://social.example/api/v1/polls/x", nil), recorder)
	err := derp.NotFound("handler", "Poll not found")

	errorHandler(err, ctx)
	errorHandler(err, ctx)

	require.JSONEq(t, `{"error":"Poll not found"}`, recorder.Body.String())
}
