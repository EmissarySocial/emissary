package templates

import (
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// renderCustomCSS renders a stylesheet the way a profile page places it: in a <style> element, inside @scope
func renderCustomCSS(t *testing.T, value string) string {

	t.Helper()

	page := template.Must(template.New("page").Funcs(FuncMap(nil)).Parse(
		`<style>@scope (#outbox-page) { {{customCSS .}} }</style><p>after</p>`,
	))

	var result strings.Builder
	require.NoError(t, page.Execute(&result, value))

	return result.String()
}

// TestCustomCSS_Registered confirms that the helper is in the FuncMap with the signature templates call
func TestCustomCSS_Registered(t *testing.T) {
	_, ok := FuncMap(nil)["customCSS"].(func(any) template.CSS)
	require.True(t, ok, "customCSS helper must be registered with this signature")
}

// TestCustomCSS_KeepsAnAcceptedStylesheet confirms that html/template writes an accepted stylesheet unescaped
func TestCustomCSS_KeepsAnAcceptedStylesheet(t *testing.T) {

	value := "nav > a::before {\n\tcontent: \"<b>\";\n}"

	require.Equal(t, template.CSS(value), customCSS(value))
	require.Equal(t, `<style>@scope (#outbox-page) { `+value+` }</style><p>after</p>`, renderCustomCSS(t, value))
}

// TestCustomCSS_RefusesABreakout is the XSS property for a helper that returns template.CSS.
//
// RULE: html/template does not escape a template.CSS value, so the helper must refuse
// anything that could end the <style> element.  See the funcmap notes in AGENTS.md.
func TestCustomCSS_RefusesABreakout(t *testing.T) {

	for _, value := range []string{
		"a {} </style><script>alert(1)</script>",
		"a {} </STYLE><script>alert(1)</script>",
		`a::before { content: "</style><script>alert(1)</script>"; }`,
		"/* </style><script>alert(1)</script> */",
	} {
		require.Empty(t, customCSS(value), "rendering %q", value)
		require.Equal(t, `<style>@scope (#outbox-page) {  }</style><p>after</p>`, renderCustomCSS(t, value), "rendering %q", value)
	}
}

// TestCustomCSS_RefusesAnEscapeFromScope confirms that a stylesheet cannot close the block that wraps it
func TestCustomCSS_RefusesAnEscapeFromScope(t *testing.T) {
	require.Empty(t, customCSS("a {} } body { display: none; }"))
	require.Empty(t, customCSS("a { color: red; "))
	require.Empty(t, customCSS(`a {} b\`))
}

// TestCustomCSS_RefusesOtherWebsites confirms that a stylesheet cannot load files from another website
func TestCustomCSS_RefusesOtherWebsites(t *testing.T) {
	require.Empty(t, customCSS("@import url(/a.css);"))
	require.Empty(t, customCSS("a { background: url(https://example.com/pixel.png); }"))
	require.Equal(t, template.CSS("a { background: url(/pixel.png); }"), customCSS("a { background: url(/pixel.png); }"))
}

// TestCustomCSS_Empty confirms that an empty stylesheet renders as nothing
func TestCustomCSS_Empty(t *testing.T) {
	require.Empty(t, customCSS(""))
}

// TestCustomCSS_MissingStylesheet verifies that a record with no stylesheet renders nothing and never fails
func TestCustomCSS_MissingStylesheet(t *testing.T) {

	require.Empty(t, customCSS(nil))
	require.Empty(t, customCSS(""))

	// A map with no "stylesheet" key hands the function a nil, as `.Data "stylesheet"` does
	page := template.Must(template.New("page").Funcs(template.FuncMap{"customCSS": customCSS}).Parse(
		`{{with customCSS (index . "stylesheet")}}<style>{{.}}</style>{{end}}ok`,
	))

	var result strings.Builder
	require.NoError(t, page.Execute(&result, map[string]any{}))
	require.Equal(t, "ok", result.String())
}
