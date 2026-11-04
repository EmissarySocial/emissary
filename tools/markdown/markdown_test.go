package markdown

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestToHTML confirms that standard Markdown syntax converts to HTML.
func TestToHTML(t *testing.T) {

	result := ToHTML("# Title\n\nHello **world** and *emphasis*.")

	require.Contains(t, result, "<h1")
	require.Contains(t, result, "Title")
	require.Contains(t, result, "<strong>world</strong>")
	require.Contains(t, result, "<em>emphasis</em>")
}

// TestToHTML_Tables confirms that the table extension is enabled.
func TestToHTML_Tables(t *testing.T) {

	result := ToHTML("| A | B |\n|---|---|\n| 1 | 2 |")

	require.Contains(t, result, "<table>")
	require.Contains(t, result, "<td>1</td>")
}

// eventHandlerAttribute matches an on* event handler sitting inside a real HTML
// tag.  Escaped text such as "&lt;svg/onload=x&gt;" is inert and must not match.
var eventHandlerAttribute = regexp.MustCompile(`(?is)<[^>]*[\s/]on[a-z]+\s*=`)

// scriptURL matches a javascript: scheme used as a URL inside a real HTML tag.
var scriptURL = regexp.MustCompile(`(?is)<[^>]*=\s*["']?\s*javascript:`)

// requireNoActiveContent asserts that an HTML fragment carries nothing the
// browser would execute: no script element, no event-handler attribute, and no
// javascript: URL.
func requireNoActiveContent(t *testing.T, html string, input string) {

	t.Helper()

	require.NotContains(t, strings.ToLower(html), "<script", "input: %q", input)
	require.NotRegexp(t, eventHandlerAttribute, html, "input: %q", input)
	require.NotRegexp(t, scriptURL, html, "input: %q", input)
}

// TestToHTML_StripsScripts confirms that raw script markup is neutralized.  Raw
// HTML is passed through the converter (WithUnsafe) and must be caught by the
// sanitizer, which either drops it or escapes it into inert text.
func TestToHTML_StripsScripts(t *testing.T) {

	closures := []string{
		"<script>alert(1)</script>",
		"Hello <script src='https://evil.example/x.js'></script>",
		"<img src=x onerror=alert(1)>",
		"<a href=\"javascript:alert(1)\">click</a>",
		"<iframe src=\"javascript:alert(1)\"></iframe>",
		"<div onclick=\"alert(1)\">click</div>",
		"<svg/onload=alert(1)>",
		"[link](javascript:alert(1))",
		"<a href=\"data:text/html;base64,PHNjcmlwdD4=\">x</a>",
	}

	for _, closure := range closures {
		requireNoActiveContent(t, ToHTML(closure), closure)
	}
}

// TestToHTML_KeepsSafeMarkup confirms that the sanitizer preserves the markup
// that the policy deliberately allows.
func TestToHTML_KeepsSafeMarkup(t *testing.T) {

	result := ToHTML(`Text <img src="https://example.com/a.png" alt="Alt"> more`)
	require.Contains(t, result, "<img")
	require.Contains(t, result, `src="https://example.com/a.png"`)
	require.Contains(t, result, `alt="Alt"`)
}

// TestToHTML_EmptyAndWhitespace confirms that degenerate inputs return empty
// output rather than erroring or panicking.
func TestToHTML_EmptyAndWhitespace(t *testing.T) {

	require.Equal(t, "", ToHTML(""))
	require.Equal(t, "", strings.TrimSpace(ToHTML("   ")))
	require.Equal(t, "", strings.TrimSpace(ToHTML("\n\n\n")))
}

// TestToHTML_AngleBrackets confirms that literal angle brackets in prose are
// escaped rather than dropped as unknown markup.
func TestToHTML_AngleBrackets(t *testing.T) {
	require.Contains(t, ToHTML("a < b"), "a &lt; b")
}

// TestToHTML_InvalidUTF8 confirms that malformed input does not panic.
func TestToHTML_InvalidUTF8(t *testing.T) {
	require.NotPanics(t, func() {
		ToHTML("# Title \xff\xfe invalid")
	})
}

// TestToHTML_LargeInput confirms that a large document converts without error.
func TestToHTML_LargeInput(t *testing.T) {

	source := strings.Repeat("# Heading\n\nSome **text** here.\n\n", 500)

	require.NotPanics(t, func() {
		require.Contains(t, ToHTML(source), "<h1")
	})
}

// TestSanitize confirms that Sanitize applies the same policy on its own, for
// HTML that did not come from Markdown.
func TestSanitize(t *testing.T) {

	require.NotContains(t, Sanitize("<script>alert(1)</script><p>Hi</p>"), "<script")
	require.Contains(t, Sanitize("<p>Hi</p>"), "<p>Hi</p>")
	require.Equal(t, "", Sanitize(""))
}

// TestSanitize_Idempotent confirms that sanitizing already-sanitized HTML does
// not change it, which matters because Content.Format sanitizes converted
// Markdown a second time.
func TestSanitize_Idempotent(t *testing.T) {

	once := ToHTML("# Title\n\n**bold** <script>alert(1)</script>\n\n| A |\n|---|\n| 1 |")
	require.Equal(t, once, Sanitize(once))
}

// TestSanitize_KeepsTextStyles confirms that an inline style may color, emphasize, and
// align text, as the Visual Editor's color picker and alignment write it.
func TestSanitize_KeepsTextStyles(t *testing.T) {

	keeps := func(input string, expected string) {
		t.Helper()
		require.Equal(t, expected, Sanitize(input), "input: %q", input)
	}

	keeps(`<span class="cdx-text-color" style="color: #0055cc;">blue</span>`, `<span class="cdx-text-color" style="color: #0055cc">blue</span>`)
	keeps(`<span style="color: rgb(0, 85, 204)">x</span>`, `<span style="color: rgb(0, 85, 204)">x</span>`)
	keeps(`<span style="color: rebeccapurple">x</span>`, `<span style="color: rebeccapurple">x</span>`)
	keeps(`<mark style="background-color: #ffff00">x</mark>`, `<mark style="background-color: #ffff00">x</mark>`)
	keeps(`<span style="font-weight: 700; font-style: italic">x</span>`, `<span style="font-weight: 700; font-style: italic">x</span>`)
	keeps(`<p style="text-align: center">x</p>`, `<p style="text-align: center">x</p>`)
	keeps(`<span style="text-decoration: underline">x</span>`, `<span style="text-decoration: underline">x</span>`)
	keeps(`<span style="text-decoration-line: line-through">x</span>`, `<span style="text-decoration-line: line-through">x</span>`)
	keeps(`<blockquote style="text-align: right">x</blockquote>`, `<blockquote style="text-align: right">x</blockquote>`)
}

// TestSanitize_DropsUnsafeStyles confirms that no declaration can load a URL, run script,
// or move content over the page, and that a safe declaration beside one still survives.
func TestSanitize_DropsUnsafeStyles(t *testing.T) {

	drops := func(input string, expected string) {
		t.Helper()
		require.Equal(t, expected, Sanitize(input), "input: %q", input)
	}

	// Properties that position, size, hide, or layer content
	drops(`<div style="position: fixed; top: 0; left: 0; z-index: 9999">x</div>`, `<div>x</div>`)
	drops(`<span style="display: none">x</span>`, `<span>x</span>`)
	drops(`<span style="width: 100vw; height: 100vh; margin: -500px">x</span>`, `<span>x</span>`)
	drops(`<span style="font-size: 900px; font-family: Comic Sans">x</span>`, `<span>x</span>`)

	// Properties and values that load a URL or run script
	drops(`<span style="background-image: url(https://tracker.example/p.gif)">x</span>`, `<span>x</span>`)
	drops(`<span style="background: url(javascript:alert(1))">x</span>`, `<span>x</span>`)
	drops(`<span style="color: url(https://tracker.example/p.gif)">x</span>`, `<span>x</span>`)
	drops(`<span style="color: expression(alert(1))">x</span>`, `<span>x</span>`)
	drops(`<span style="behavior: url(x.htc); -moz-binding: url(x.xml)">x</span>`, `<span>x</span>`)
	drops(`<span style="color: red; background-color: url(javascript:alert(1))">x</span>`, `<span style="color: red">x</span>`)

	// Declarations that are not CSS at all
	drops(`<span style="color">x</span>`, `<span>x</span>`)
	drops(`<span style=";;;">x</span>`, `<span>x</span>`)
	drops(`<span style="color: red; position: absolute">x</span>`, `<span style="color: red">x</span>`)
	drops(`<span style="color: red&quot; onmouseover=&quot;alert(1)">x</span>`, `<span>x</span>`)
}

// TestSanitize_StyleIdempotent confirms that a sanitized style passes a second sanitize unchanged.
func TestSanitize_StyleIdempotent(t *testing.T) {

	// Stream.Save runs the html-emissary schema format over Content.Format's output
	once := Sanitize(`<p style="text-align: center"><span class="cdx-text-color" style="color: #0055cc; position: fixed">x</span></p>`)
	require.Equal(t, `<p style="text-align: center"><span class="cdx-text-color" style="color: #0055cc">x</span></p>`, once)
	require.Equal(t, once, Sanitize(once))
}

// styleAttribute matches the value of every style attribute in sanitized output
var styleAttribute = regexp.MustCompile(`(?is)\sstyle="([^"]*)"`)

// unsafeStyle matches anything in a style value that could load a URL, run script, or move content
var unsafeStyle = regexp.MustCompile(`(?i)url\s*\(|expression\s*\(|javascript|position|z-index|display|width|height|margin|binding|behavior|import|\\`)

// FuzzSanitize_Styles confirms that no style attribute survives carrying a URL, script, or layout
func FuzzSanitize_Styles(f *testing.F) {

	f.Add(`<span style="color: #0055cc">x</span>`)
	f.Add(`<span style="color: burlywood">x</span>`)
	f.Add(`<span style="color: red; position: fixed">x</span>`)
	f.Add(`<span style="background: url(javascript:alert(1))">x</span>`)
	f.Add(`<span style="color: expression(alert(1))">x</span>`)
	f.Add(`<span style="color: \75 rl(x)">x</span>`)
	f.Add(`<span style="color: red&quot; onmouseover=&quot;alert(1)">x</span>`)
	f.Add(`<p style="text-align: center; text-decoration: underline wavy red">x</p>`)
	f.Add("<span style=\"color:\xff\">x</span>")

	f.Fuzz(func(t *testing.T, input string) {

		result := Sanitize(input)
		requireNoActiveContent(t, result, input)

		for _, match := range styleAttribute.FindAllStringSubmatch(result, -1) {
			require.NotRegexp(t, unsafeStyle, match[1], "input: %q", input)
		}
	})
}

// FuzzToHTML confirms that no input can panic the converter, and that its output
// never carries a script tag through the sanitizer.
func FuzzToHTML(f *testing.F) {

	f.Add("")
	f.Add("# Title")
	f.Add("<script>alert(1)</script>")
	f.Add("a < b")
	f.Add("| A | B |\n|---|---|\n| 1 | 2 |")
	f.Add("[link](javascript:alert(1))")
	f.Add("\xff\xfe")
	f.Add("![img](x\" onerror=\"alert(1))")

	f.Fuzz(func(t *testing.T, source string) {
		requireNoActiveContent(t, ToHTML(source), source)
	})
}
