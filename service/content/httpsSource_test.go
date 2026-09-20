package content

import (
	"fmt"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
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

// TestContentFormat pins the Markdown-or-HTML decision against addresses measured live on
// GitHub, Codeberg, GitLab, Gitea and cgit (2026-09-20).  All five serve a raw file as
// text/plain whatever it holds, which is why the extension decides and the header does not.
func TestContentFormat(t *testing.T) {

	tests := []struct {
		name        string
		url         string
		contentType string
		expected    string
	}{
		// Raw endpoints, as the four forges actually answer them
		{"github raw md", "https://raw.githubusercontent.com/golang/go/master/README.md", "text/plain; charset=utf-8", model.ContentFormatMarkdown},
		{"codeberg raw md", "https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/README.md", "text/plain; charset=utf-8", model.ContentFormatMarkdown},
		{"gitlab raw md", "https://gitlab.com/gitlab-org/gitlab/-/raw/master/README.md", "text/plain; charset=utf-8", model.ContentFormatMarkdown},
		{"gitea raw md", "https://gitea.com/gitea/tea/raw/branch/main/README.md", "text/plain; charset=utf-8", model.ContentFormatMarkdown},

		// A raw .html file is also served as text/plain, so the extension is the only signal
		{"github raw html", "https://raw.githubusercontent.com/h5bp/html5-boilerplate/main/src/index.html", "text/plain; charset=utf-8", model.ContentFormatHTML},

		// RULE: a Markdown extension outranks the header.  A forge's file PAGE lands here, and
		// is read as Markdown rather than refused -- the page's own markup becomes the body.
		{"github blob page", "https://github.com/golang/go/blob/master/README.md", "text/html; charset=utf-8", model.ContentFormatMarkdown},
		{"declared markdown, named html", "https://example.com/docs/page.html", "text/markdown", model.ContentFormatHTML},

		// RULE: the query string is not part of the extension.  cgit serves this exact shape.
		{"cgit md with query", "https://git.kernel.org/pub/scm/git/git.git/plain/README.md?h=master", "text/plain; charset=UTF-8", model.ContentFormatMarkdown},
		{"cgit html with query", "https://git.kernel.org/pub/scm/git/git.git/plain/index.html?h=master", "text/plain; charset=UTF-8", model.ContentFormatHTML},

		// No extension at all: the header is all that is left
		{"no extension, plain", "https://example.com/docs/guide", "text/plain", model.ContentFormatMarkdown},
		{"no extension, html", "https://example.com/docs/guide", "text/html", model.ContentFormatHTML},
		{"no extension, xhtml", "https://example.com/docs/guide", "application/xhtml+xml", model.ContentFormatHTML},

		// Spelling variants on both sides
		{"long markdown extension", "https://example.com/a.markdown", "text/html", model.ContentFormatMarkdown},
		{"short html extension", "https://example.com/a.htm", "text/plain", model.ContentFormatHTML},
		{"uppercase extension", "https://example.com/A.MD", "text/plain", model.ContentFormatMarkdown},
		{"uppercase media type", "https://example.com/a.txt", "TEXT/PLAIN", model.ContentFormatMarkdown},
		{"x-markdown", "https://example.com/a.txt", "text/x-markdown", model.ContentFormatMarkdown},

		// An unrelated text extension carries no opinion, so text/plain means Markdown
		{"other extension", "https://example.com/notes.txt", "text/plain", model.ContentFormatMarkdown},
	}

	for _, test := range tests {
		result, err := contentFormat(test.url, test.contentType)
		require.NoError(t, err, test.name)
		require.Equal(t, test.expected, result, test.name)
	}
}

// TestContentFormat_Refuses pins the allowlist.  A mistyped address pointing at an image or an
// archive is refused rather than stored as somebody's article.
func TestContentFormat_Refuses(t *testing.T) {

	refuses := []struct {
		name        string
		contentType string
	}{
		{"json", "application/json"},
		{"octet stream", "application/octet-stream"},
		{"png", "image/png"},
		{"zip", "application/zip"},
		{"unparseable", "text/plain; charset="},
		{"absent", ""},
	}

	for _, test := range refuses {
		// The .md extension cannot rescue a media type this package will not read
		_, err := contentFormat("https://example.com/docs/page.md", test.contentType)
		require.Error(t, err, test.name)
		require.True(t, derp.IsClientError(err), "%s: an unusable source is the author's to fix", test.name)
	}
}

// TestSourceExtension pins that the PATH is read and the query string is not
func TestSourceExtension(t *testing.T) {

	tests := map[string]string{
		"https://example.com/a.md":               ".md",
		"https://example.com/a.MD":               ".md",
		"https://example.com/a.md?h=master":      ".md",
		"https://example.com/a.html?v=2#heading": ".html",
		"https://example.com/docs/guide":         "",
		"https://example.com/":                   "",
		"https://example.com/a.b.markdown":       ".markdown",
		"https://example.com/dir.md/file":        "",
	}

	for url, expected := range tests {
		require.Equal(t, expected, sourceExtension(url), url)
	}
}
