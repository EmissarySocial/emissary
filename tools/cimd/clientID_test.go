package cimd

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestParseClientID_Accepts verifies the client_ids IndieAuth §3.3 allows, and the canonical form of each
func TestParseClientID_Accepts(t *testing.T) {

	accepts := func(value string, expected string) {
		t.Helper()
		actual, err := ParseClientID(value)
		require.NoError(t, err, "parsing %q", value)
		require.Equal(t, expected, actual, "parsing %q", value)
	}

	accepts("https://fed.brid.gy/", "https://fed.brid.gy/")
	accepts("https://app.example.com/client.json", "https://app.example.com/client.json")
	accepts("http://app.example.com/", "http://app.example.com/")
	accepts("https://app.example.com:8443/client", "https://app.example.com:8443/client")
	accepts("https://app.example.com/client?version=2", "https://app.example.com/client?version=2")
	accepts("http://localhost:8080/client.json", "http://localhost:8080/client.json")

	// A URL with no path is read as "/" (IndieAuth §3.4)
	accepts("https://fed.brid.gy", "https://fed.brid.gy/")

	// The scheme and host are case-insensitive, so they are lower-cased; the path is not
	accepts("HTTPS://Fed.Brid.GY/Path", "https://fed.brid.gy/Path")
}

// TestParseClientID_Refuses verifies that every client_id IndieAuth §3.3 forbids is refused as a bad request
func TestParseClientID_Refuses(t *testing.T) {

	refuses := func(value string, reason string) {
		t.Helper()
		_, err := ParseClientID(value)
		require.Error(t, err, "%s: %q", reason, value)
		require.True(t, derp.IsBadRequest(err), "%s: %q answers %d", reason, value, derp.ErrorCode(err))
	}

	refuses("", "empty")
	refuses("fed.brid.gy", "no scheme")
	refuses("/client.json", "relative")
	refuses("ftp://app.example.com/", "wrong scheme")
	refuses("javascript:alert(1)", "opaque")
	refuses("https:app.example.com", "opaque with an http scheme")
	refuses("https://app.example.com/#", "empty fragment")
	refuses("https://app.example.com/#section", "fragment")
	refuses("https://user@app.example.com/", "username")
	refuses("https://user:secret@app.example.com/", "username and password")
	refuses("https://127.0.0.1/", "IPv4 address")
	refuses("https://[::1]/", "IPv6 address")
	refuses("https://10.0.0.1:8080/client", "private IPv4 address")
	refuses("https:///client.json", "no host")
	refuses("https://not_a_domain/", "invalid host")
	refuses("https://app.example.com/./client", "single-dot segment")
	refuses("https://app.example.com/a/../client", "double-dot segment")
	refuses("https://app.example.com/%2e%2e/client", "encoded double-dot segment")
	refuses("https://app.example.com/"+strings.Repeat("a", maxClientIDLength), "too long")
	refuses("https://app.example.com/\x00", "control character")
}

// FuzzParseClientID verifies that ParseClientID never panics, and that a canonical client_id is its own canonical form
func FuzzParseClientID(f *testing.F) {

	for _, seed := range []string{
		"https://fed.brid.gy/",
		"https://fed.brid.gy",
		"HTTPS://Fed.Brid.GY/Path?x=1",
		"http://localhost:8080/client.json",
		"https://127.0.0.1/",
		"https://user@app.example.com/",
		"https://app.example.com/a/../client",
		"https://app.example.com/#",
		"javascript:alert(1)",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {

		canonical, err := ParseClientID(value)

		if err != nil {
			require.Empty(t, canonical)
			return
		}

		// Parsing is idempotent
		again, err := ParseClientID(canonical)
		require.NoError(t, err, "canonical form %q of %q is refused", canonical, value)
		require.Equal(t, canonical, again)

		// The properties every accepted client_id has
		require.True(t, strings.HasPrefix(canonical, "http://") || strings.HasPrefix(canonical, "https://"))
		require.NotContains(t, canonical, "#")
	})
}
