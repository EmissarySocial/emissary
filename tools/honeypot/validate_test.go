package honeypot

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateValues verifies that a populated decoy field is rejected and an empty one is not
func TestValidateValues(t *testing.T) {

	table := []struct {
		name     string
		values   url.Values
		rejected bool
	}{
		{"empty decoy passes", url.Values{"phoneNumber": {""}}, false},
		{"absent decoy passes", url.Values{"name": {"Sarah"}}, false},
		{"populated decoy is rejected", url.Values{"phoneNumber": {"555-1234"}}, true},
		{"one of several is enough", url.Values{"address1": {""}, "phoneNumber": {"x"}}, true},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {

			err := ValidateValues(testCase.values, "phoneNumber", "address1")

			if testCase.rejected {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestValidate_BodyIsReReadable verifies that Validate leaves a body every later consumer can
// still read. ReadRequestBody restores a ONE-SHOT reader, so without the re.Reader swap the
// next consumer drains it and every one after that sees an empty form -- which, for a honeypot
// running beside a form parser, means silently passing every request.
func TestValidate_BodyIsReReadable(t *testing.T) {

	const body = "name=Sarah&phoneNumber="

	request, err := http.NewRequest("POST", "/", strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	require.NoError(t, Validate(request, "phoneNumber"))

	// Three consecutive reads must all see the full body
	for attempt := 1; attempt <= 3; attempt++ {
		again, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.Equal(t, body, string(again), "read %d saw a drained body", attempt)
	}
}

// TestValidate_RejectsPopulatedField verifies the request-level wrapper still rejects
func TestValidate_RejectsPopulatedField(t *testing.T) {

	request, err := http.NewRequest("POST", "/", strings.NewReader("phoneNumber=555-1234"))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	require.Error(t, Validate(request, "phoneNumber"))
}
