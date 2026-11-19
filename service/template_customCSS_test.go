package service

import (
	"html/template"
	"strings"
	"testing"
	"testing/fstest"

	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/stretchr/testify/require"
)

// customCSSPage is the line that places a profile's stylesheet, as qwertylicious' outbox.html writes it
const customCSSPage = `{{- with customCSS . }}<style>@scope (#outbox-page) { {{.}} }</style>{{- end }}<p>after</p>`

// renderCustomCSSPage loads customCSSPage the way the server loads a template folder, minifier
// included, then renders it with the provided stylesheet
func renderCustomCSSPage(t *testing.T, stylesheet string) string {

	t.Helper()

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	pageTemplate := template.New("page").Funcs(funcMap)
	filesystem := fstest.MapFS{"page.html": &fstest.MapFile{Data: []byte(customCSSPage)}}

	require.NoError(t, loadHTMLTemplateFromFilesystem(filesystem, pageTemplate, funcMap))

	var buffer strings.Builder
	require.NoError(t, pageTemplate.ExecuteTemplate(&buffer, "page", stylesheet))

	return buffer.String()
}

// TestCustomCSS_SurvivesTheLoader confirms that the minifier leaves the <style> element and its
// action alone, and that html/template writes an accepted stylesheet unescaped
func TestCustomCSS_SurvivesTheLoader(t *testing.T) {

	stylesheet := "nav > a::before {\n\tcontent: \"<b>\";\n}"

	require.Equal(t, `<style>@scope (#outbox-page) { `+stylesheet+` }</style><p>after</p>`, renderCustomCSSPage(t, stylesheet))
}

// TestCustomCSS_LoaderDropsARefusedStylesheet confirms that a refused or empty stylesheet writes no
// <style> element at all, so nothing it held reaches the page
func TestCustomCSS_LoaderDropsARefusedStylesheet(t *testing.T) {

	for _, stylesheet := range []string{
		"",
		"a {} </style><script>alert(1)</script>",
		"a {} } body { display: none; }",
		"a { background: url(https://example.com/pixel.png); }",
	} {
		require.Equal(t, `<p>after</p>`, renderCustomCSSPage(t, stylesheet), "rendering %q", stylesheet)
	}
}
