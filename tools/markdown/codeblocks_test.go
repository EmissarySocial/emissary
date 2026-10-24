package markdown

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// codeClass matches every class attribute on a <code> element in the output
var codeClass = regexp.MustCompile(`<code[^>]*class="([^"]*)"`)

// eventHandler matches an on* attribute inside any tag in the output
var eventHandler = regexp.MustCompile(`(?i)<[^>]*\son[a-z]+\s*=`)

// TestToHTML_CodeBlock_KeepsLanguage confirms that a fenced block names its
// language with a class that browser-side highlighters can read.
func TestToHTML_CodeBlock_KeepsLanguage(t *testing.T) {

	result := ToHTML("```go\nx := 1\n```\n")

	require.Equal(t, "<pre><code class=\"language-go\">x := 1\n</code></pre>\n", result)
}

// TestToHTML_CodeBlock_NoLanguage confirms that a fenced block without a
// language, and an indented block, carry no class at all.
func TestToHTML_CodeBlock_NoLanguage(t *testing.T) {

	require.Equal(t, "<pre><code>plain\n</code></pre>\n", ToHTML("```\nplain\n```\n"))
	require.Equal(t, "<pre><code>indented\n</code></pre>\n", ToHTML("    indented\n"))
}

// TestToHTML_CodeBlock_NoHighlightMarkup confirms that the server adds no
// highlighting markup of its own: no spans, no inline styles.
func TestToHTML_CodeBlock_NoHighlightMarkup(t *testing.T) {

	result := ToHTML("```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```\n")

	require.NotContains(t, result, "<span")
	require.NotContains(t, result, "style=")
	require.Contains(t, result, "func main() {\n\tprintln(&#34;hi&#34;)\n}")
}

// TestToHTML_CodeBlock_Escapes confirms that markup inside a code block is
// shown as text, not rendered.
func TestToHTML_CodeBlock_Escapes(t *testing.T) {

	result := ToHTML("```html\n<script>alert(1)</script>\n```\n")

	require.Contains(t, result, `class="language-html"`)
	require.Contains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
	require.NotContains(t, result, "<script")
}

// TestToHTML_CodeBlock_InfoStrings pins how the sanitizer treats unusual
// fence info strings, which authors control.
func TestToHTML_CodeBlock_InfoStrings(t *testing.T) {

	check := func(fence string, expectedClass string) {
		t.Helper()
		result := ToHTML("```" + fence + "\ncode\n```\n")
		matches := codeClass.FindStringSubmatch(result)

		if expectedClass == "" {
			require.Nil(t, matches, "fence %q gave %s", fence, result)
			return
		}

		require.NotNil(t, matches, "fence %q gave %s", fence, result)
		require.Equal(t, expectedClass, matches[1], "fence %q", fence)
	}

	// Only the first word of the info string names the language
	check("go {.numberLines}", "language-go")
	check("TypeScript", "language-TypeScript")
	check("hjson", "language-hjson")
	check("objective-c", "language-objective-c")

	// The sanitizer drops a class holding characters outside letters, digits, "_" and "-"
	check("c++", "")

	// Attribute injection through the info string never escapes the class
	check(`go" onclick="alert(1)`, "")
	check(`go"><script>`, "")
}

// FuzzToHTML_CodeBlock confirms that no fence info string or body can
// produce a script, an event handler, or a code class outside language-*.
func FuzzToHTML_CodeBlock(f *testing.F) {

	f.Add("go", "x := 1")
	f.Add("", "plain")
	f.Add(`go" onclick="alert(1)`, "x")
	f.Add("html", "<script>alert(1)</script>")
	f.Add("c++", "int main() {}")
	f.Add("objective-c", "@end")
	f.Add("\x00\xff", "\xff")
	f.Add("\f", "0")
	f.Add(" onCliCk=`", "0")

	// The sanitizer's class rule (bluemonday SpaceSeparatedTokens).  Whitespace can
	// appear: goldmark splits the info string on spaces only, so a form feed stays
	allowedClass := regexp.MustCompile(`^language-[\s\p{L}\p{N}_-]*$`)

	f.Fuzz(func(t *testing.T, info string, body string) {

		result := ToHTML("```" + info + "\n" + body + "\n```\n")
		if strings.Contains(strings.ToLower(result), "<script") {
			t.Fatalf("script in output: %q", result)
		}

		// Only an attribute inside a tag is a handler; escaped text may say anything
		if eventHandler.MatchString(result) {
			t.Fatalf("event handler in output: %q", result)
		}

		for _, match := range codeClass.FindAllStringSubmatch(result, -1) {
			if !allowedClass.MatchString(match[1]) {
				t.Fatalf("unexpected code class %q in output: %q", match[1], result)
			}
		}
	})
}
