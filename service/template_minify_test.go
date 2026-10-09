package service

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// templateAction matches a single Go template action, including a multi-line one
var templateAction = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

// htmlComment matches an HTML comment, which the minifier deletes outright
var htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)

// TestEmbeddedTemplates_ActionsSurviveMinification asserts that minifying a template does not
// rewrite the Go actions inside it.
//
// Templates are minified BEFORE they are parsed, and the minifier does not know Go template
// syntax.  Inside a tag it reads whatever it finds as attributes and lowercases it, so
//
//	<input value="X"{{if eq $w "MEDIUM"}} checked{{end}}>
//
// silently becomes `eq $w "medium"` -- valid syntax that can never match.  Inside an attribute
// VALUE it re-balances quotes, so `value="{{.QueryParam "username"}}"` becomes
// `.QueryParam " username"`, which looks up a parameter that does not exist.  Neither failure
// raises an error at startup or at render time; the page simply comes out wrong.
//
// A third variant is loudest but still worth naming: an action inside an element that admits
// only specific children -- `<select>`, which admits only `<option>` -- is read as a stray text
// node and DELETED, taking a whole conditional with it.
//
// The safe forms are: put the action outside the tag entirely (emit the whole tag, or the whole
// restricted element, from inside the branch); hoist an in-tag value into an all-lowercase
// variable declared in element content, where its case survives; and use backticks for any
// string argument that appears inside an attribute value.
//
// This supersedes an earlier TestEmbeddedTemplates_MinifierPreservesCase, which matched actions
// by their lowercase form and so could detect only the first variant.
func TestEmbeddedTemplates_ActionsSurviveMinification(t *testing.T) {

	// Configured exactly as loadHTMLTemplateFromFilesystem configures it
	m := minify.New()
	minifier := html.Minifier{KeepEndTags: true, KeepQuotes: true, KeepDocumentTags: true}
	m.AddFunc("text/html", minifier.Minify)

	root := "../_embed/templates"
	checked := 0

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {

		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}

		raw, err := os.ReadFile(path) // nolint:gosec // test-only, fixed root
		require.NoError(t, err, "read %s", path)

		// Drop HTML comments first.  The minifier deletes them, so an action quoted inside
		// one as documentation would otherwise read as an action that went missing.
		source := htmlComment.ReplaceAllString(string(raw), "")

		minified, err := m.String("text/html", source)
		require.NoError(t, err, "minify %s", path)

		before := templateAction.FindAllString(source, -1)
		after := templateAction.FindAllString(minified, -1)

		require.Equal(t, len(before), len(after), "%s: minifying changed the NUMBER of template actions", path)

		for index := range before {
			if isTemplateComment(before[index]) {
				continue // a {{/* */}} comment is free to reflow; it renders nothing
			}

			require.Equal(t, normalizeAction(before[index]), normalizeAction(after[index]),
				"%s: minifying rewrote a template action.  Move it out of the tag, or use "+
					"backticks for its string arguments.", path)
		}

		checked++
		return nil
	})

	require.NoError(t, err)
	require.Positive(t, checked, "no HTML templates found under "+root)
}

// isTemplateComment returns TRUE if an action is a Go template comment
func isTemplateComment(action string) bool {
	return strings.HasPrefix(strings.TrimLeft(strings.TrimPrefix(action, "{{"), "- "), "/*")
}

// normalizeAction collapses each run of whitespace in an action to a single space, and trims
// the padding just inside the delimiters.  Runs are collapsed because the minifier reflows
// long actions harmlessly; WHERE the whitespace sits is preserved, because a space that moves
// into a quoted string is exactly the silent breakage this test exists to catch.
func normalizeAction(action string) string {
	action = strings.TrimSuffix(strings.TrimPrefix(action, "{{"), "}}")
	action = strings.TrimSuffix(strings.TrimPrefix(action, "-"), "-")
	return strings.Join(strings.Fields(action), " ")
}
