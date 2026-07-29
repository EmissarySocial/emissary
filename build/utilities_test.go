package build

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

func TestParseForm(t *testing.T) {

	body := strings.NewReader(`first=1&second=2&third=3&third=4`)

	request, err := http.NewRequest("POST", "http://test", io.NopCloser(body))
	require.Nil(t, err)

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	values, err := formdata.Parse(request)
	require.Nil(t, err)
	require.Equal(t, []string{"1"}, values["first"])
}

func TestParseMultipartForm(t *testing.T) {

	request, err := getTestRequest()
	require.Nil(t, err)

	values, err := formdata.Parse(request)
	require.Nil(t, err)

	require.Equal(t, []string{"https://amazon.com"}, values["data.links.AMAZON"])
	require.Equal(t, []string{"WEBHOOK", "OTHER"}, values["syndication"])
	require.Equal(t, []string{"CC-BY"}, values["data.license"])
	require.Equal(t, []string{"http://localhost/6692c69bfe80a9aacf125b0d/attachments/6723b7b74aa88ca07dc8614e"}, values["iconUrl"])
}

func TestIsNewOrEmpty(t *testing.T) {
	require.True(t, isNewOrEmpty(""))
	require.True(t, isNewOrEmpty("new"))
	require.True(t, isNewOrEmpty("NEW"))
	require.False(t, isNewOrEmpty("1234567890abcdef12345678"))
}

// getTestRequest mocks an HTTP request with a multipart form body
func getTestRequest() (*http.Request, error) {

	// Here's the body of the request.  Note two values for "ima_slice"
	body := `POST /6692c69bfe80a9aacf125b0d/edit HTTP/1.1
Content-Type: multipart/form-data; boundary=----WebKitFormBoundaryVfyfBnHAjBwnl9dd
Accept: */*
Sec-Fetch-Site: same-origin
Accept-Language: en-US,en;q=0.9
Accept-Encoding: gzip, deflate
Sec-Fetch-Mode: cors
Origin: http://localhost
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1.1 Safari/605.1.15
Content-Length: 3611
Referer: http://localhost/@a11y
Connection: keep-alive
Sec-Fetch-Dest: empty
Cookie: Authorization=eyJhbGciOiJIUzI1NiIsImtpZCI6IjIwMjUwMTA2IiwidHlwIjoiSldUIn0.eyJVIjoiNjY5MmJiMzg4NjZiMjczZDc0Y2UxY2UyIiwiRyI6W10sIkMiOiIwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAiLCJleHAiOjIwNTE3MzEwNTUsImlhdCI6MTczNjE5ODI1NX0.9ZsTcfGCJJjPKiP_ypQYjeu9BtauMOVHOfyDPVc7aOI; Authorization-backup=eyJhbGciOiJIUzI1NiIsImtpZCI6IjIwMjUwMTA2IiwidHlwIjoiSldUIn0.eyJVIjoiNjY5MDU4OTQ3ZjMwZTRiZjk0YTIwNjU2IiwiRyI6W10sIkMiOiIwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAiLCJPIjp0cnVlLCJleHAiOjIwNTE3Mjg5OTcsImlhdCI6MTczNjE5NjE5N30.kvrq-vXmKmuZROPNz73NYL5DT1XlVLL-DrQITdricrQ
HX-Request: true
Priority: u=3, i
HX-Current-URL: http://localhost/@a11y


------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="label"

Def Album Jam
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.year"

2024
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.license"

CC-BY
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="iconUrl"

http://localhost/6692c69bfe80a9aacf125b0d/attachments/6723b7b74aa88ca07dc8614e
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="isFeatured"

true
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.tags"

#all #rock #funky
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="summary"

Notes notes notes.  Lots and lots of notes.

Package regexp implements regular expression search.

The syntax of the regular expressions accepted is the same general syntax used by Perl, Python, and other languages. More precisely, it is the syntax accepted by RE2 and described at https://golang.org/s/re2syntax, except for \C. For an overview of the syntax, see the regexp/syntax package.

The regexp implementation provided by this package is guaranteed to run in time linear in the size of the input. (This is a property not guaranteed by most open source implementations of regular expressions.) For more information about this property, see https://swtch.com/~rsc/regexp/regexp1.html or any book about automata theory.

All characters are UTF-8-encoded code points. Following utf8.DecodeRune, each byte of an invalid UTF-8 sequence is treated as if it encoded utf8.RuneError (U+FFFD).

There are 16 methods of Regexp that match a regular expression and identify the matched text. Their names are matched by this regular expression:
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.AMAZON"

https://amazon.com
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.APPLE"

https://apple.com
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.GOOGLE"


------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.SOUNDCLOUD"


------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.SPOTIFY"


------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.TIDAL"


------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.YOUTUBE"


------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.OTHER1"

https://why-would-you-have-a-url-that-is-this-long.oh-well.social
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.OTHER2"

https://thereallylongnandnamegoeshere.bandcamp.com
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.links.OTHER3"


------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.color.body"

#f19a64
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.color.page"

#f1dfc6
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="data.color.button"

#e34522
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="syndication"

WEBHOOK
------WebKitFormBoundaryVfyfBnHAjBwnl9dd
Content-Disposition: form-data; name="syndication"

OTHER
------WebKitFormBoundaryVfyfBnHAjBwnl9dd--
`

	// Create a new HTTP request
	reader := io.NopCloser(strings.NewReader(body))
	bufferedReader := bufio.NewReader(reader)
	result, err := http.ReadRequest(bufferedReader)
	return result, err
}

// TestOEmbedURL proves the permalink survives as a single query-string VALUE.
// Raw concatenation used to truncate it at the first "&" or "#".
func TestOEmbedURL(t *testing.T) {

	// A plain permalink round-trips unchanged
	{
		result := oEmbedURL("https://bandwagon.fm", "https://bandwagon.fm/@681c4330b949b6b581af8a51", "json")

		parsed, err := url.Parse(result)

		require.Nil(t, err)
		require.Equal(t, "/.oembed", parsed.Path)
		require.Equal(t, "https://bandwagon.fm/@681c4330b949b6b581af8a51", parsed.Query().Get("url"))
		require.Equal(t, "json", parsed.Query().Get("format"))
	}

	// A permalink containing reserved characters is not truncated
	{
		permalink := "https://bandwagon.fm/my-stream?a=1&b=2#anchor"
		result := oEmbedURL("https://bandwagon.fm", permalink, "xml")

		parsed, err := url.Parse(result)

		require.Nil(t, err)
		require.Equal(t, permalink, parsed.Query().Get("url"))
		require.Equal(t, "xml", parsed.Query().Get("format"))
	}

	// The port in a development permalink is preserved
	{
		permalink := "http://localhost:8080/@681c4330b949b6b581af8a51"
		result := oEmbedURL("http://localhost:8080", permalink, "json")

		parsed, err := url.Parse(result)

		require.Nil(t, err)
		require.Equal(t, permalink, parsed.Query().Get("url"))
	}
}

func TestInlineErrorMessage(t *testing.T) {

	// A validation error buried under pipeline wrappers surfaces its ROOT message --
	// the regression this pins: users saw "Executing steps for child" instead.
	{
		err := derp.Validation("Address not found", "@ghost@nowhere.invalid")
		wrapped := derp.Wrap(derp.Wrap(derp.Wrap(err, "location", "Resolving actor address"), "location", "Saving model object"), "location", "Executing steps for child")

		require.True(t, derp.IsValidationError(wrapped))
		require.Equal(t, "Address not found", inlineErrorMessage(wrapped))
	}

	// A cause passed to derp.Validation as a DETAIL is not part of the unwrap chain,
	// so the friendly message -- not the transport error -- is still the root.
	{
		cause := derp.Internal("remote.Transaction", "connection refused")
		err := derp.Wrap(derp.Validation("Address not found", "http://bob.local:8082/@abc", cause), "location", "Executing steps for child")

		require.Equal(t, "Address not found", inlineErrorMessage(err))
	}

	// A NON-validation error keeps its outermost message: the root of these chains is
	// internals (database/network guts) and the outer wrap is the presentable text.
	{
		cause := derp.Internal("mongo.Session", "no reachable servers")
		err := derp.Wrap(cause, "location", "Loading user")

		require.False(t, derp.IsValidationError(err))
		require.Equal(t, "Loading user", inlineErrorMessage(err))
	}

	// An unwrapped validation error returns its own message
	{
		err := derp.Validation("Current password is incorrect")
		require.Equal(t, "Current password is incorrect", inlineErrorMessage(err))
	}
}

func TestWrapInlineError(t *testing.T) {

	// The full writer path: 200 status, htmx retargeting headers, and the root
	// validation message inside the red span.
	recorder := httptest.NewRecorder()
	err := derp.Wrap(derp.Validation("Address not found"), "location", "Executing steps for child")

	require.Nil(t, WrapInlineError(recorder, err))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "innerHTML", recorder.Header().Get("HX-Reswap"))
	require.Equal(t, "#htmx-response-message", recorder.Header().Get("HX-Retarget"))
	require.Equal(t, `<span class="text-red">Address not found</span>`, recorder.Body.String())

	// A message echoing hostile input is escaped, not swapped into the page as markup
	recorder = httptest.NewRecorder()
	err = derp.Validation(`<script>alert(1)</script> is not a valid address`)

	require.Nil(t, WrapInlineError(recorder, err))
	require.Equal(t, `<span class="text-red">&lt;script&gt;alert(1)&lt;/script&gt; is not a valid address</span>`, recorder.Body.String())
}
