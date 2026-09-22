package service

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/icon"
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

// TestEmbeddedTemplates_HTMLParses parses each *.html file in the directories listed below exactly
// the way the server does at startup (minify → html/template Parse with the real funcMap).  This
// catches template syntax errors, an undefined funcMap function, and the html/template
// attribute-context gotcha where a "-quoted string literal inside a "-quoted HTML attribute breaks
// the lexer — without having to boot the server.
//
// It is NOT a full sweep, and the `dirs` list below is the whole of its coverage: a template
// directory that is not named there is never parsed by any test, and its first parse happens in
// front of a visitor.
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

	// A full sweep trips over pre-existing email/layout templates that legitimately rely on
	// cross-file variables like $title and therefore do not parse standalone.  Every directory
	// that CAN parse standalone belongs here — nothing else parses these files before a visitor
	// asks for one.
	dirs := []string{"user-inbox", "user-settings", "theme-default", "stream-article-base", "stream-article-remote"}

	for _, name := range dirs {
		dir := filepath.Join(root, name)
		files, err := os.ReadDir(dir)
		require.NoError(t, err, "read dir %s", dir)

		parseDir(t, dir, files)
	}
}

// compile-time assertion that our stub satisfies the interface
var _ icon.Provider = nullIconProvider{}
