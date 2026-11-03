package formdata

import (
	"mime"
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

// TestParse_AcceptsMediaTypeParameters verifies that parameters after the media type, and its
// letter case, never refuse a form.  fetch sends the first header below for a URLSearchParams body.
func TestParse_AcceptsMediaTypeParameters(t *testing.T) {

	accepts := func(contentType string) {
		t.Helper()

		request := httptest.NewRequest(http.MethodPost, "/submit?page=2", strings.NewReader("content=Hello+there"))
		request.Header.Set("Content-Type", contentType)

		values, err := Parse(request)
		require.NoError(t, err, "Content-Type %q", contentType)
		require.Equal(t, "Hello there", values.Get("content"), "Content-Type %q", contentType)
		require.Equal(t, "2", values.Get("page"), "Content-Type %q", contentType)

		body, err := ParseBody(request)
		require.NoError(t, err, "Content-Type %q", contentType)
		require.Equal(t, "Hello there", body.Get("content"), "Content-Type %q", contentType)
		require.Empty(t, body.Get("page"), "Content-Type %q", contentType)
	}

	accepts("application/x-www-form-urlencoded;charset=UTF-8")
	accepts("application/x-www-form-urlencoded; charset=utf-8")
	accepts("application/x-www-form-urlencoded ; charset=\"UTF-8\"")
	accepts("Application/X-WWW-Form-URLEncoded; Charset=UTF-8")
	accepts("application/x-www-form-urlencoded; charset=UTF-8; extra=value")
}

// TestParse_AcceptsMultipartParameters verifies that a multipart form is read whatever parameters
// travel beside its boundary, and in either letter case
func TestParse_AcceptsMultipartParameters(t *testing.T) {

	request := multipartRequest(t, "", map[string]string{"name": "Sarah"})
	_, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	require.NoError(t, err)

	request.Header.Set("Content-Type", "Multipart/Form-Data; charset=UTF-8; boundary="+params["boundary"])

	values, err := ParseBody(request)
	require.NoError(t, err)
	require.Equal(t, "Sarah", values.Get("name"))
}

// TestParse_RejectsMalformedContentType verifies that a header which is not a form's media type,
// or that cannot be parsed at all, is refused rather than read as a form
func TestParse_RejectsMalformedContentType(t *testing.T) {

	refuses := func(contentType string) {
		t.Helper()

		request := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("content=Hello"))
		request.Header.Set("Content-Type", contentType)

		values, err := Parse(request)
		require.Error(t, err, "Content-Type %q", contentType)
		require.Empty(t, values, "Content-Type %q", contentType)

		_, err = ParseBody(request)
		require.Error(t, err, "Content-Type %q", contentType)
	}

	refuses("")
	refuses("text/plain; charset=UTF-8")
	refuses("application/x-www-form-urlencoded; charset")
	refuses("application/x-www-form-urlencoded-extra")
	refuses("multipart/form-dataX; boundary=abc")
	refuses("; charset=UTF-8")
}

// TestParse_RejectsUndecodableBody verifies that a form body which cannot be decoded is reported,
// in both encodings, rather than read as a partial form
func TestParse_RejectsUndecodableBody(t *testing.T) {

	urlencoded := urlencodedRequest("", "content=%zz")
	_, err := Parse(urlencoded)
	require.Error(t, err, "an invalid percent-escape")

	multipartBody := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("not a multipart body"))
	multipartBody.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	_, err = ParseBody(multipartBody)
	require.Error(t, err, "a body with no boundary")
}

// FuzzParse_ContentType verifies that any Content-Type header either parses or is refused,
// never panics, and is accepted only for one of the two form media types
func FuzzParse_ContentType(f *testing.F) {

	f.Add("application/x-www-form-urlencoded")
	f.Add("application/x-www-form-urlencoded;charset=UTF-8")
	f.Add("Application/X-WWW-Form-URLEncoded; Charset=UTF-8")
	f.Add("multipart/form-data; boundary=abc")
	f.Add("application/x-www-form-urlencoded; charset")
	f.Add("multipart/form-dataX; boundary=abc")
	f.Add("")
	f.Add("\x00;;==\"")

	f.Fuzz(func(t *testing.T, contentType string) {

		request := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("content=Hello"))
		request.Header.Set("Content-Type", contentType)

		_, err := Parse(request)

		if err != nil {
			return
		}

		// RULE: an accepted header names one of the two form media types
		mediaType, _, parseErr := mime.ParseMediaType(contentType)
		require.NoError(t, parseErr, "accepted Content-Type %q", contentType)
		require.Contains(t, []string{"application/x-www-form-urlencoded", "multipart/form-data"}, mediaType, "accepted Content-Type %q", contentType)
	})
}
