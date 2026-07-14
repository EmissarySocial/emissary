package model

import (
	"bytes"
	htmltemplate "html/template"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEmail_BodyAutoEscapes is a regression test for CWE-79/CWE-116: the HTML email body
// must be rendered with html/template so that user- or remote-controlled values are
// contextually auto-escaped. If the Body template ever reverts to text/template, the
// injected <script> below would render raw and this test would fail.
func TestEmail_BodyAutoEscapes(t *testing.T) {

	email := NewEmail("test", testEmailFuncMap())

	bodyTemplate, err := email.Body.Parse(`{{icon "bell"}}|{{.Title}}`)
	require.NoError(t, err)
	email.Body = bodyTemplate

	var buffer bytes.Buffer
	err = email.Body.Execute(&buffer, map[string]any{"Title": `<script>alert(1)</script>`})
	require.NoError(t, err)

	output := buffer.String()

	// The attacker-controlled value must be escaped, not rendered as live markup.
	require.NotContains(t, output, "<script>")
	require.Contains(t, output, "&lt;script&gt;alert(1)&lt;/script&gt;")

	// A helper that returns template.HTML must still pass through un-escaped, so that
	// legitimate template markup (icons, sanitized content) continues to render.
	require.Contains(t, output, "<svg>bell</svg>")
}

// TestEmail_PlainTextFieldsAreNotHTMLEscaped confirms the To/Subject/Headers templates stay
// text/template. These are plain-text contexts, and HTML-escaping them would corrupt values
// (e.g. turn "&" in a subject line into "&amp;", or entity-mangle an email address).
func TestEmail_PlainTextFieldsAreNotHTMLEscaped(t *testing.T) {

	email := NewEmail("test", testEmailFuncMap())

	subjectTemplate, err := email.Subject.Parse(`Hello {{.Name}}`)
	require.NoError(t, err)
	email.Subject = subjectTemplate

	var buffer bytes.Buffer
	err = email.Subject.Execute(&buffer, map[string]any{"Name": "A & B <C>"})
	require.NoError(t, err)

	// Plain-text context: the raw value is preserved verbatim (no HTML entity encoding).
	require.Equal(t, "Hello A & B <C>", buffer.String())
}

// testEmailFuncMap returns a minimal funcMap whose "icon" helper returns template.HTML,
// mirroring the real funcMap so the tests can verify template.HTML pass-through behavior.
func testEmailFuncMap() htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"icon": func(name string) htmltemplate.HTML {
			return htmltemplate.HTML("<svg>" + name + "</svg>")
		},
	}
}
