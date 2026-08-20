package service

import (
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/icon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// nullIconProvider is a no-op icon.Provider so the funcMap's icon() helper resolves during parsing.
type nullIconProvider struct{}

// Get implements the icon.Provider interface. The stub renders nothing.
func (nullIconProvider) Get(name string) string { return "" }

// Write implements the icon.Provider interface. The stub writes nothing.
func (nullIconProvider) Write(name string, writer io.Writer) {}

// TestEmbeddedTemplates_HTMLParses walks every _embed/templates directory and parses each *.html
// file exactly the way the server does at startup (minify → html/template Parse with the real
// funcMap).  This catches template syntax errors — including the html/template attribute-context
// gotcha where a "-quoted string literal inside a "-quoted HTML attribute breaks the lexer — without
// having to boot the server.
func TestEmbeddedTemplates_HTMLParses(t *testing.T) {

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})

	m := minify.New()
	minifier := html.Minifier{KeepEndTags: true, KeepQuotes: true, KeepDocumentTags: true}
	m.AddFunc("text/html", minifier.Minify)

	root := "../_embed/templates"

	// Parse all HTML files in a single directory into one parse-tree (siblings define each other).
	parseDir := func(t *testing.T, dir string, files []os.DirEntry) {
		tree := template.New(dir).Funcs(funcMap)

		for _, file := range files {
			name := file.Name()
			if !strings.HasSuffix(name, ".html") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(dir, name)) // nolint:gosec // test-only, fixed root
			require.NoError(t, err, "read %s/%s", dir, name)

			contentString := string(content)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			actionID := strings.TrimSuffix(name, ".html")
			_, err = tree.New(actionID).Parse(contentString)
			require.NoError(t, err, "parse template %s/%s", dir, name)
		}
	}

	// Scope to the template directories touched by the notifications feature.  (A full sweep trips
	// over pre-existing email/layout templates that legitimately rely on cross-file variables like
	// $title and therefore do not parse standalone.)
	dirs := []string{"user-inbox", "user-settings", "theme-default"}

	for _, name := range dirs {
		dir := filepath.Join(root, name)
		files, err := os.ReadDir(dir)
		require.NoError(t, err, "read dir %s", dir)

		parseDir(t, dir, files)
	}
}

// templateActionRegex matches a single {{...}} template action, including trim markers.
var templateActionRegex = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

// TestEmbeddedTemplates_MinifierPreservesCase guards a silent, execution-time-only trap: the HTML
// minifier that every template passes through at startup lowercases *attribute names*, and it treats
// a {{...}} action sitting in attribute position (between the tag name and the closing ">") as an
// attribute.  So `<a {{if $x.IsConversation}}...>` reaches html/template as `.isconversation`, which
// parses fine and then fails with "can't evaluate field" the first time the page is rendered.
//
// Actions inside quoted attribute *values* and inside element content are left alone -- only the
// in-tag ones are rewritten.  The fix at each call site is to hoist the value into an all-lowercase
// variable in element content, where the case survives.
func TestEmbeddedTemplates_MinifierPreservesCase(t *testing.T) {

	m := minify.New()
	minifier := html.Minifier{KeepEndTags: true, KeepQuotes: true, KeepDocumentTags: true}
	m.AddFunc("text/html", minifier.Minify)

	err := filepath.Walk("../_embed/templates", func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := os.ReadFile(path) // nolint:gosec // test-only, fixed root
		require.NoError(t, err, "read %s", path)

		minified, err := m.String("text/html", string(content))
		require.NoError(t, err, "minify %s", path)

		assertActionsPreserveCase(t, path, string(content), minified)

		return nil
	})

	require.NoError(t, err, "walking template directory")
}

// assertActionsPreserveCase fails the test for every {{...}} action that the minifier rewrote into a
// different case.  `original` and `minified` are the same template before and after minification.
func assertActionsPreserveCase(t *testing.T, path string, original string, minified string) {

	t.Helper()

	// Count each original action so that duplicates are matched one-for-one.  The two lists cannot be
	// compared by index, because the minifier also *removes* actions (those inside HTML comments) and
	// collapses the whitespace around the ones it keeps.
	counts := make(map[string]int)
	byLowercase := make(map[string][]string)

	for _, action := range templateActionRegex.FindAllString(original, -1) {
		counts[action]++
		lowercase := strings.ToLower(action)
		byLowercase[lowercase] = append(byLowercase[lowercase], action)
	}

	// An action that survived minification unchanged is fine.  One with no exact match, but that does
	// match an original case-insensitively, was rewritten.
	for _, action := range templateActionRegex.FindAllString(minified, -1) {

		if counts[action] > 0 {
			counts[action]--
			continue
		}

		for _, source := range byLowercase[strings.ToLower(action)] {

			if source == action {
				continue
			}

			assert.Fail(t,
				"minifier lowercased a template action",
				"%s\n  before: %s\n  after:  %s\n\nHoist this value into an all-lowercase variable outside the tag.",
				path, strings.TrimSpace(source), strings.TrimSpace(action))

			break
		}
	}
}

// compile-time assertion that our stub satisfies the interface
var _ icon.Provider = nullIconProvider{}
