package content

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestParseSourceURL_Accepts pins the addresses a Stream author may supply
func TestParseSourceURL_Accepts(t *testing.T) {

	accepts := func(value string, allowPrivateIPs bool) {
		t.Helper()
		result, err := parseSourceURL(value, allowPrivateIPs)
		require.NoError(t, err, value)
		require.NotEmpty(t, result, value)
	}

	accepts("https://raw.githubusercontent.com/org/repo/main/docs/page.md", false)
	accepts("https://gitlab.com/ns/repo/-/raw/main/README.md", false)
	accepts("https://codeberg.org/o/r/raw/branch/main/README.md", false)
	accepts("https://example.com/page.md?ref=main#section", false)
	accepts("https://example.com:8443/page.md", false)

	// Plain HTTP is for local development, where private addresses are already allowed
	accepts("http://localhost:8080/page.md", true)
}

// TestParseSourceURL_Refuses pins every address that never reaches the network
func TestParseSourceURL_Refuses(t *testing.T) {

	refuses := func(name string, value string, allowPrivateIPs bool) {
		t.Run(name, func(t *testing.T) {
			_, err := parseSourceURL(value, allowPrivateIPs)
			require.Error(t, err)
			require.True(t, derp.IsClientError(err), "a bad address is the author's to fix")
		})
	}

	// RULE: Credentials never live in the address, where they would be stored in plain text
	refuses("username", "https://user@example.com/page.md", false)
	refuses("username and password", "https://user:secret@example.com/page.md", false)

	refuses("plain http", "http://example.com/page.md", false)
	refuses("file scheme", "file:///etc/passwd", false)
	refuses("file scheme, private allowed", "file:///etc/passwd", true)
	refuses("git scheme", "git://example.com/repo.git", false)
	refuses("ssh scheme", "ssh://git@example.com/repo.git", false)
	refuses("javascript scheme", "javascript:alert(1)", false)
	refuses("data scheme", "data:text/plain,hello", false)
	refuses("no scheme", "example.com/page.md", false)
	refuses("no host", "https:///page.md", false)
	refuses("empty", "", false)
	refuses("invalid port", "https://example.com:notaport/page.md", false)
	refuses("control character", "https://exa\x7fmple.com/page.md", false)
	refuses("too long", "https://example.com/"+strings.Repeat("a", maxURLLength), false)
}

// TestParseSourceURL_PasswordNeverEchoed confirms a refusal cannot leak what it refused
func TestParseSourceURL_PasswordNeverEchoed(t *testing.T) {

	_, err := parseSourceURL("https://user:hunter2@example.com/page.md", false)

	require.Error(t, err)
	require.NotContains(t, derp.Message(err), "hunter2")
	require.NotContains(t, fmt.Sprint(derp.Details(err)...), "hunter2")
}
