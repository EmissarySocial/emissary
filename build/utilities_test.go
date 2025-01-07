package build

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/tools/formdata"
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
