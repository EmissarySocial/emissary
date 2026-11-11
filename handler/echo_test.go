package handler

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEchoHeaderValue confirms a header that carries a credential is never printed, in any letter case
func TestEchoHeaderValue(t *testing.T) {

	for _, name := range []string{"Authorization", "authorization", "Cookie", "COOKIE", "Proxy-Authorization"} {
		require.Equal(t, echoRedacted, echoHeaderValue(name, "secret-value"), name)
	}

	require.Equal(t, "text/html", echoHeaderValue("Accept", "text/html"))
	require.Equal(t, "10.0.0.1", echoHeaderValue("X-Forwarded-For", "10.0.0.1"))
}

// TestEchoBody confirms a body is written as sorted form values, as plain lines, or as "(empty)"
func TestEchoBody(t *testing.T) {

	write := func(contentType string, body string) string {
		var buffer bytes.Buffer
		echoBody(&buffer, contentType, []byte(body))
		return buffer.String()
	}

	require.Equal(t, "  (empty)\n", write("text/plain", ""))
	require.Equal(t, "  one\n  two\n", write("text/plain", "one\ntwo"))
	require.Equal(t, "  a:                   1\n  b:                   2\n", write("application/x-www-form-urlencoded", "b=2&a=1"))

	// A form that does not parse is written as it arrived
	require.Equal(t, "  a=%zz\n", write("application/x-www-form-urlencoded", "a=%zz"))
}

// TestEchoSortedKeys confirms names come back in alphabetical order, and none from an empty set
func TestEchoSortedKeys(t *testing.T) {
	require.Equal(t, []string{"a", "b", "c"}, echoSortedKeys(map[string][]string{"c": nil, "a": nil, "b": nil}))
	require.Empty(t, echoSortedKeys(nil))
}
