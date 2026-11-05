package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/stretchr/testify/require"
)

// hostileShareText closes a quoted hyperscript string, so it runs as script anywhere it is pasted inside one
const hostileShareText = `x'}); alert(1); ({a:'`

// socialOutlineBuilder is the smallest dot that base-social's btn-social-outline renders against
type socialOutlineBuilder struct {
	signedIn bool
}

// IsAuthenticated returns TRUE when the simulated visitor is signed in here
func (builder socialOutlineBuilder) IsAuthenticated() bool {
	return builder.signedIn
}

// GetResponseSummary returns a summary with no Like or Boost, for any URL
func (builder socialOutlineBuilder) GetResponseSummary(string) model.UserResponseSummary {
	return model.NewUserResponseSummary()
}

// BasePath returns the simulated Stream's path
func (builder socialOutlineBuilder) BasePath() string {
	return "/0123456789abcdef01234567"
}

// StreamID returns the simulated Stream's ID
func (builder socialOutlineBuilder) StreamID() string {
	return "0123456789abcdef01234567"
}

// Permalink returns the simulated Stream's ActivityPub ID
func (builder socialOutlineBuilder) Permalink() string {
	return "https://example.social/0123456789abcdef01234567"
}

// shareWidgetStub stands in for the Widget builder, supplying the Stream accessors that the Share Button reads
type shareWidgetStub struct {
	url     string
	label   string
	summary string
}

// URL returns the simulated Stream's URL
func (stub shareWidgetStub) URL() string {
	return stub.url
}

// Label returns the simulated Stream's label
func (stub shareWidgetStub) Label() string {
	return stub.label
}

// Summary returns the simulated Stream's summary
func (stub shareWidgetStub) Summary() string {
	return stub.summary
}

// renderBaseSocial executes one base-social HTML template against the provided dot
func renderBaseSocial(t *testing.T, file string, dot any) string {

	t.Helper()

	templateService := loadProfileTabsTemplates(t, nil)
	baseSocial, exists := templateService.templatePrep["base-social"]
	require.True(t, exists, "base-social is not loaded")

	var buffer strings.Builder
	require.NoError(t, baseSocial.HTMLTemplate.ExecuteTemplate(&buffer, file, dot))

	return buffer.String()
}

// renderShareWidget loads the shipped Share Button widget and renders it against the stub
func renderShareWidget(t *testing.T, stub shareWidgetStub) string {

	t.Helper()

	const directory = "widget-share"

	widgetService := NewWidget(emissarytemplates.FuncMap(nullIconProvider{}))
	filesystem := os.DirFS(filepath.Join("../_embed/templates", directory))
	definitionType, definition := findDefinition(filesystem)
	require.Equal(t, DefinitionWidget, definitionType)
	require.NoError(t, widgetService.Add(directory, filesystem, definition))

	widget, exists := widgetService.Get("share")
	require.True(t, exists, "the widget must register under the id declared in widget.hjson")

	var buffer strings.Builder
	require.NoError(t, widget.HTMLTemplate.ExecuteTemplate(&buffer, "widget", stub))

	return buffer.String()
}

// TestShareButton_SentHidden confirms that share-button carries its URL in an attribute and waits for the Share behavior
func TestShareButton_SentHidden(t *testing.T) {

	html := renderBaseSocial(t, "share-button", "https://example.social/0123456789abcdef01234567")

	require.Equal(t, `<button type="button" script="install Share" data-share-url="https://example.social/0123456789abcdef01234567" hidden><span aria-hidden="true" class="margin-left-xs"></span> Share</button>`, strings.TrimSpace(html))
}

// TestShareButton_URLContext confirms that the URL is escaped as a URL, so no remote post can share a script
func TestShareButton_URLContext(t *testing.T) {

	html := renderBaseSocial(t, "share-button", "javascript:alert(1)")
	require.Contains(t, html, `data-share-url="#ZgotmplZ"`)

	html = renderBaseSocial(t, "share-button", `https://example.social/a"b c`)
	require.Contains(t, html, `data-share-url="https://example.social/a%22b%20c"`)
}

// TestSocialOutline_ShareIsLast confirms that both rows end with Share, inside the span that a signed-in POST replaces
func TestSocialOutline_ShareIsLast(t *testing.T) {

	// RULE: A signed-in POST swaps the span, so Share must be inside it or each Like or Boost adds a copy
	html := strings.TrimSpace(renderBaseSocial(t, "btn-social-outline", socialOutlineBuilder{signedIn: true}))
	require.Equal(t, 1, strings.Count(html, `script="install Share"`))
	require.Regexp(t, `<button type="button" class="outline" script="install Share" data-share-url="https://example.social/0123456789abcdef01234567" hidden><span aria-hidden="true" class="margin-left-xs"></span> Share</button>\s*</span>$`, html)

	// A visitor who is not signed in gets the same Share at the end of the intent buttons
	html = strings.TrimSpace(renderBaseSocial(t, "btn-social-outline", socialOutlineBuilder{}))
	require.Equal(t, 1, strings.Count(html, `script="install Share"`))
	require.Regexp(t, `<button type="button" class="outline" script="install Share" data-share-url="https://example.social/0123456789abcdef01234567" hidden>\s*<span class="show-desktop">Share</span>\s*</button>$`, html)
	require.Less(t, strings.Index(html, "intent=like"), strings.Index(html, "install Share"))
}

// TestShareWidget_NoTextInScript confirms that a Stream's label and summary never reach the hyperscript
func TestShareWidget_NoTextInScript(t *testing.T) {

	html := renderShareWidget(t, shareWidgetStub{
		url:     "https://example.social/@alice/my-page",
		label:   hostileShareText,
		summary: hostileShareText,
	})

	// RULE: The script is fixed, and the text travels in attributes that hyperscript only reads
	require.Contains(t, html, `script="install Share"`)
	require.NotContains(t, html, "navigator.share")
	require.Equal(t, 1, strings.Count(html, `data-share-title="x&#39;}); alert(1); ({a:&#39;"`))
	require.Equal(t, 1, strings.Count(html, `data-share-text="x&#39;}); alert(1); ({a:&#39;"`))
	require.Contains(t, html, `data-share-url="https://example.social/@alice/my-page"`)

	// The Share behavior shows the button only where the browser has an OS share menu
	require.Contains(t, html, " hidden>")
	require.Contains(t, html, "Share Page</div>")
}
