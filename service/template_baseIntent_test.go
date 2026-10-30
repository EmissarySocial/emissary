package service

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// intentModalBuilder stands in for the Builder that renders base-intent's modal, which reads only these two values
type intentModalBuilder struct {
	query url.Values
}

// QueryParam returns one query parameter of the simulated request
func (builder intentModalBuilder) QueryParam(name string) string {
	return builder.query.Get(name)
}

// BasePath returns the simulated path of the page that opened the modal
func (builder intentModalBuilder) BasePath() string {
	return "/@alice"
}

// renderIntentModal minifies, parses, and executes base-intent/intent.html the way the server does, for one query string
func renderIntentModal(t *testing.T, query string) string {

	t.Helper()

	content, err := os.ReadFile("../_embed/templates/base-intent/intent.html")
	require.NoError(t, err)

	m := minify.New()
	minifier := html.Minifier{KeepEndTags: true, KeepQuotes: true, KeepDocumentTags: true}
	m.AddFunc("text/html", minifier.Minify)

	minified, err := m.String("text/html", string(content))
	require.NoError(t, err)

	tmpl, err := template.New("intent").Funcs(testFuncMap()).Parse(minified)
	require.NoError(t, err)

	values, err := url.ParseQuery(query)
	require.NoError(t, err)

	var buffer bytes.Buffer
	require.NoError(t, tmpl.Execute(&buffer, intentModalBuilder{query: values}))
	return buffer.String()
}

// TestBaseIntentModal_Like verifies that the intent name is lowercased and the object reaches CamperJS
func TestBaseIntentModal_Like(t *testing.T) {

	result := renderIntentModal(t, "intent=Like&object=https%3A%2F%2Fexample.social%2Fnote")

	require.Contains(t, result, `data-camper-intent="like"`)
	require.Contains(t, result, `data-camper-object="https://example.social/note"`)
	require.Contains(t, result, `data-camper-proxy-url="/.proxy"`)
	require.Contains(t, result, `hx-on:camper:hide=`)
	require.Contains(t, result, `src="/.themes/global/resources/camperjs-0.2.0/camper.js"`)
	require.NotContains(t, result, "/follow-email")
}

// TestEmbeddedTemplates_LoadVersionedCamperJS verifies that no embedded template loads the unversioned CamperJS bundle
func TestEmbeddedTemplates_LoadVersionedCamperJS(t *testing.T) {

	// RULE: Resources are cached for 30 days (handler/resources.go), so markup that needs a new bundle must
	// name a new path. resources/camperjs/ holds the bundle from before the data-camper-* contract.
	root := "../_embed/templates"

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		// The bundles themselves are not templates
		if entry.IsDir() || strings.Contains(path, "theme-global/resources/") {
			return nil
		}

		content, err := os.ReadFile(path) // #nosec G304 G122 -- a test walking the checked-in templates folder
		require.NoError(t, err)
		require.NotContains(t, string(content), "/resources/camperjs/", path)
		return nil
	})

	require.NoError(t, err)
}

// TestBaseIntentModal_CreateReply verifies that a Create answering a post becomes a Reply to that post
func TestBaseIntentModal_CreateReply(t *testing.T) {

	result := renderIntentModal(t, "intent=create&inReplyTo=https%3A%2F%2Fexample.social%2Fnote")

	require.Contains(t, result, `data-camper-intent="reply"`)
	require.Contains(t, result, `data-camper-object="https://example.social/note"`)
	require.Contains(t, result, `data-camper-inreplyto="https://example.social/note"`)
}

// TestBaseIntentModal_CreateShare verifies that a plain Create stays a Create and carries its content
func TestBaseIntentModal_CreateShare(t *testing.T) {

	result := renderIntentModal(t, "intent=create&content=Hello+world")

	require.Contains(t, result, `data-camper-intent="create"`)
	require.Contains(t, result, `data-camper-content="Hello world"`)
}

// TestBaseIntentModal_Follow verifies that only the Follow intent offers email updates
func TestBaseIntentModal_Follow(t *testing.T) {

	result := renderIntentModal(t, "intent=follow&object=https%3A%2F%2Fexample.social%2F%40alice")

	require.Contains(t, result, `data-camper-intent="follow"`)
	require.Contains(t, result, `hx-get="/@alice/follow-email"`)
}

// TestBaseIntentModal_Manage verifies that a modal opened with no intent lists accounts, and a click chooses the default
func TestBaseIntentModal_Manage(t *testing.T) {

	result := renderIntentModal(t, "")

	require.Contains(t, result, "Your Fediverse Account")
	require.Contains(t, result, `data-camper-account-action="select-account"`)
	require.Contains(t, result, `hx-on:camper:select-account="htmx.trigger(window, 'closeModal')"`, "choosing an account closes the manager")
	require.Contains(t, result, `data-camper-proxy-url="/.proxy"`)
	require.NotContains(t, result, "data-camper-intent=")
}

// TestBaseIntentModal_NoAccountLayout verifies that a visitor with no saved account meets the Fediverse first, as in the Follow modal
func TestBaseIntentModal_NoAccountLayout(t *testing.T) {

	for _, query := range []string{"intent=like&object=x", ""} {

		result := renderIntentModal(t, query)

		// The introduction and the join link hide once CamperJS finds a saved account
		require.Regexp(t, `<div class="camper-hide-if-has-accounts">\s*<div>\s*The Fediverse is an online community`, result, query)
		require.Contains(t, result, `href="https://joinmastodon.org"`, query)
		require.Contains(t, result, `<h2 class="camper-hide-if-has-accounts">Already Connected?</h2>`, query)
		require.NotContains(t, result, "emissary.social/hosting", "the old host-finding section is gone")
		require.NotContains(t, result, "max-accounts", "a visitor keeps one account, so nothing declares a limit")

		// The order follows the Follow modal: introduction, join, Already Connected, then the form
		intro := strings.Index(result, "The Fediverse is an online community")
		join := strings.Index(result, "Join the Fediverse")
		connected := strings.Index(result, "Already Connected?")
		form := strings.Index(result, `class="camper-form`)
		require.True(t, (intro < join) && (join < connected) && (connected < form), query)
	}
}

// TestBaseIntentModal_Invalid verifies that an unknown intent draws no account widgets
func TestBaseIntentModal_Invalid(t *testing.T) {

	for _, query := range []string{"intent=bogus", "intent=%22%3E%3Cscript%3E"} {
		result := renderIntentModal(t, query)

		require.Contains(t, result, "Invalid Intent", query)
		require.NotContains(t, result, "camper-accounts", query)
		require.NotContains(t, result, "<script>", query)
		require.NotContains(t, result, "joinmastodon", query)
	}
}

// TestBaseIntentModal_EscapesParameters verifies that a hostile query value cannot leave its attribute
func TestBaseIntentModal_EscapesParameters(t *testing.T) {

	result := renderIntentModal(t, "intent=like&object=%22%3E%3Cimg+src%3Dx+onerror%3Dalert(1)%3E")

	require.NotContains(t, result, "<img")
	require.Contains(t, result, `data-camper-object="&#34;&gt;&lt;img src=x onerror=alert(1)&gt;"`)
}
