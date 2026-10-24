package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestAPINotFound confirms an unmatched API path gets a JSON 404 while a registered route still wins.
func TestAPINotFound(t *testing.T) {

	// Register one real route alongside the catch-all, as server.go does
	e := echo.New()
	e.GET("/api/v1/instance", func(ctx echo.Context) error { return ctx.NoContent(http.StatusOK) })
	e.GET("/api/**", APINotFound)
	e.HEAD("/api/**", APINotFound)

	// An unknown path falls through to the catch-all
	unknown := httptest.NewRecorder()
	e.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/api/v1/timelines/link", nil))
	require.Equal(t, http.StatusNotFound, unknown.Code)
	require.Contains(t, unknown.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)
	require.JSONEq(t, `{"error":"Not found"}`, unknown.Body.String())

	// HEAD answers the same status with no body
	head := httptest.NewRecorder()
	e.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/api/v1/zzz", nil))
	require.Equal(t, http.StatusNotFound, head.Code)

	// A registered route beats the wildcard
	known := httptest.NewRecorder()
	e.ServeHTTP(known, httptest.NewRequest(http.MethodGet, "/api/v1/instance", nil))
	require.Equal(t, http.StatusOK, known.Code)
}

// TestGetTranslationLanguages confirms the translation language list is an empty JSON object
func TestGetTranslationLanguages(t *testing.T) {

	e := echo.New()
	e.GET("/api/v1/instance/translation_languages", GetTranslationLanguages)
	e.GET("/api/**", APINotFound)

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/instance/translation_languages", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)
	require.JSONEq(t, `{}`, recorder.Body.String())
}
