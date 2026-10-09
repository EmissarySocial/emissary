package formdata

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// urlencodedRequest returns a POST carrying the provided body and query string
func urlencodedRequest(query string, body string) *http.Request {

	request := httptest.NewRequest(http.MethodPost, "/submit?"+query, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return request
}

// multipartRequest returns a multipart POST carrying the provided fields and query string
func multipartRequest(t *testing.T, query string, fields map[string]string) *http.Request {

	t.Helper()

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	for name, value := range fields {
		require.NoError(t, writer.WriteField(name, value))
	}

	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/submit?"+query, strings.NewReader(body.String()))
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

// TestParse_MergesQueryString verifies that Parse merges the query string into its result.
// Callers such as build.StepEditRegistration read a control parameter this way, so the merge
// is deliberate and must not be "fixed" -- ParseBody is the entry point that excludes it.
func TestParse_MergesQueryString(t *testing.T) {

	request := urlencodedRequest("registrationId=abc123", "name=Sarah")

	values, err := Parse(request)

	require.NoError(t, err)
	require.Equal(t, "Sarah", values.Get("name"))
	require.Equal(t, "abc123", values.Get("registrationId"), "Parse must expose query parameters")
}

// TestParseBody_IgnoresQueryString verifies that ParseBody returns only what the body carried.
// A field named in BOTH places yields both values from Parse, so a consumer that joins them
// would emit attacker-controlled text appended to the visitor's own.
func TestParseBody_IgnoresQueryString(t *testing.T) {

	request := urlencodedRequest("email=attacker@evil.com&note=injected", "email=sarah@example.com&message=Hello")

	// Parse sees the merge -- both values, under one name
	merged, err := Parse(request)
	require.NoError(t, err)
	require.Equal(t, []string{"sarah@example.com", "attacker@evil.com"}, merged["email"])
	require.Equal(t, "injected", merged.Get("note"))

	// ParseBody sees only the body
	values, err := ParseBody(request)
	require.NoError(t, err)
	require.Equal(t, []string{"sarah@example.com"}, values["email"], "the query value must not survive")
	require.Equal(t, "Hello", values.Get("message"))
	require.Empty(t, values["note"], "a name that appears ONLY in the query must not appear at all")
}

// TestParseBody_Multipart verifies that ParseBody excludes the query string for multipart
// submissions too. ParseMultipartForm populates PostForm alongside Form (Go issue 9305), so
// both encodings behave the same way here.
func TestParseBody_Multipart(t *testing.T) {

	request := multipartRequest(t, "message=injected", map[string]string{
		"name":    "Sarah",
		"message": "Hello",
	})

	values, err := ParseBody(request)

	require.NoError(t, err)
	require.Equal(t, "Sarah", values.Get("name"))
	require.Equal(t, []string{"Hello"}, values["message"], "the query value must not survive")
}

// TestParseBody_NoBody verifies that a request with nothing to parse returns an empty,
// non-nil set rather than something a caller must nil-check
func TestParseBody_NoBody(t *testing.T) {

	request := urlencodedRequest("name=fromquery", "")

	values, err := ParseBody(request)

	require.NoError(t, err)
	require.NotNil(t, values)
	require.Empty(t, values.Get("name"))
}

// TestParse_RejectsUnsupportedEncoding verifies that an unrecognized Content-Type is refused
// by both entry points, rather than silently returning nothing
func TestParse_RejectsUnsupportedEncoding(t *testing.T) {

	request := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(`{"name":"Sarah"}`))
	request.Header.Set("Content-Type", "application/json")

	_, err := Parse(request)
	require.Error(t, err)

	_, err = ParseBody(request)
	require.Error(t, err)
}
