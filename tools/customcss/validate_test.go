package customcss

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestValidate_Accepts confirms that ordinary hand-written CSS passes every rule unchanged
func TestValidate_Accepts(t *testing.T) {

	accept := func(name string, value string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.NoError(t, Validate(value), "validating %q", value)
		})
	}

	accept("empty", "")
	accept("whitespace only", " \n\t ")
	accept("one rule", "a { color: red; }")
	accept("child combinator", "nav > a { color: red; }")
	accept("several lines", "#profile-card {\n\tborder: 2px solid red;\n}\n\nh1 { font-size: 2rem; }\n")
	accept("CRLF line endings", "a {\r\n\tcolor: red;\r\n}\r\n")
	accept("quoted content with markup", `a::before { content: "<b>"; }`)
	accept("braces inside a string", `a::before { content: "}{"; }`)
	accept("closing parenthesis inside a string", `a::before { content: ")"; }`)
	accept("escaped quote inside a string", `a::before { content: "say \"hi\""; }`)
	accept("single-quoted string", `a::before { content: 'x'; }`)
	accept("braces inside a comment", "/* } */ a { color: red; }")
	accept("empty comment", "/**/a{}")
	accept("media range query", "@media (width < 600px) { a { color: red; } }")
	accept("nested rules", "a { & b { color: red; } }")
	accept("keyframes", "@keyframes spin { from { rotate: 0deg; } to { rotate: 360deg; } }")
	accept("font-face from this website", `@font-face { font-family: X; src: url(/fonts/x.woff2) format("woff2"); }`)
	accept("relative url", "a { background: url(images/a.png); }")
	accept("root-relative quoted url", `a { background: url("/@me/attachments/1"); }`)
	accept("url with spaces inside the parentheses", "a { background: url(  /a.png  ); }")
	accept("empty url", "a { background: url(); }")
	accept("uppercase URL", "a { background: URL(/a.png); }")
	accept("fragment url", "a { filter: url(#blur); }")
	accept("data url", `a { background: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg'%3E%3C/svg%3E"); }`)
	accept("image-set from this website", `a { background: image-set("/a.png" 1x, "/b.png" 2x); }`)
	accept("type() inside image-set", `a { background: image-set("/a.avif" type("image/avif")); }`)
	accept("var outside a loading function", `a { --x: 4px; margin: var(--x); color: var(--c, "red"); }`)
	accept("theme resource", `a { background: url(/.themes/global/resources/a.png); }`)
	accept("an address as plain text", `a::after { content: "https://example.com"; }`)
	accept("escaped class name", `.md\:flex { display: flex; }`)
	accept("escaped backslash at the end", `a {} b\\`)
	accept("custom property holding a block", "a { --theme: { color: red; }; }")
	accept("scope root", ":scope { padding: 1rem; }")
	accept("nested @scope", "@scope (.card) { img { border: 0; } }")
	accept("@import as plain text", `a::after { content: "@import"; }`)
	accept("attribute selector", `a[href^="https://"] { color: red; }`)
	accept("charset", `@charset "utf-8";`)
	accept("unicode", "a::after { content: \"✓ café\"; }")
	accept("longest allowed", "a{}"+strings.Repeat(" ", maxLength-3))
}

// TestValidate_Refuses confirms that each rule refuses what it exists to refuse, and names the line
func TestValidate_Refuses(t *testing.T) {

	refuse := func(name string, value string, message string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			err := Validate(value)
			require.Error(t, err, "validating %q", value)
			require.True(t, derp.IsValidationError(err), "validating %q", value)
			require.Equal(t, message, derp.Message(err), "validating %q", value)
		})
	}

	// Breaking out of the <style> element
	refuse("closing style tag", "a { color: red; } </style><script>alert(1)</script>", `The stylesheet cannot contain "</" (line 1).`)
	refuse("closing tag inside a string", `a::before { content: "</style>"; }`, `The stylesheet cannot contain "</" (line 1).`)
	refuse("closing tag inside a comment", "/* </style> */", `The stylesheet cannot contain "</" (line 1).`)
	refuse("closing tag on a later line", "a {}\n\n</b", `The stylesheet cannot contain "</" (line 3).`)
	refuse("data url with markup", `a { background: url("data:image/svg+xml,<svg></svg>"); }`, `The stylesheet cannot contain "</" (line 1).`)

	// Breaking out of the @scope that wraps it
	refuse("extra closing brace", "a { color: red; } } body { display: none; }", `The stylesheet has a '}' with nothing to close (line 1).`)
	refuse("closing brace first", "} body { display: none; }", `The stylesheet has a '}' with nothing to close (line 1).`)
	refuse("brace closes a function", "a { color: rgb(1 2 3 }", `The stylesheet has a '}' where it needs a ')' (line 1).`)
	refuse("brace closes a bracket", "a { b[href } ]", `The stylesheet has a '}' where it needs a ']' (line 1).`)
	refuse("block inside a bracket stays open", "a[href { }", `The stylesheet never closes the '[' that it opens on line 1.`)
	refuse("stray parenthesis", "a ) { }", `The stylesheet has a ')' with nothing to close (line 1).`)
	refuse("stray bracket", "a ] { }", `The stylesheet has a ']' with nothing to close (line 1).`)
	refuse("unclosed block", "a {\n\tcolor: red;", `The stylesheet never closes the '{' that it opens on line 1.`)
	refuse("unclosed inner block", "a {\n\tb {\n}", `The stylesheet never closes the '{' that it opens on line 1.`)
	refuse("unclosed function", "a { }\nb { color: rgb(1 2 3", `The stylesheet never closes the '(' that it opens on line 2.`)
	refuse("unclosed bracket", "a[href", `The stylesheet never closes the '[' that it opens on line 1.`)
	refuse("final backslash", "a {}\n0\\", `The stylesheet ends with a backslash (line 2).`)
	refuse("three final backslashes", `a {} b\\\`, `The stylesheet ends with a backslash (line 1).`)

	// Strings, comments, and addresses that a browser could read differently
	refuse("unterminated string", `a::before { content: "abc`, `The stylesheet has a string that never ends (line 1).`)
	refuse("escaped final quote", `a::before { content: "abc\"`, `The stylesheet has a string that never ends (line 1).`)
	refuse("string across lines", "a::before { content: \"abc\n\"; }", `The stylesheet has a string that runs onto a new line (line 1).`)
	refuse("unterminated comment", "a {}\n/* forever", `The stylesheet has a comment that never ends (line 2).`)
	refuse("unterminated url", "a { background: url(/a.png", `The stylesheet has a url() that never ends (line 1).`)
	refuse("unterminated quoted url", `a { background: url("/a.png`, `The stylesheet has a url() that never ends (line 1).`)
	refuse("escaped quote ends a quoted url", `a { background: url("/a.png\")`, `The stylesheet has a url() that never ends (line 1).`)
	refuse("escaped parenthesis ends a url", `url(\)`, `The stylesheet has a url() that never ends (line 1).`)
	refuse("space inside an unquoted url", "a { background: url(/a b.png); }", `The stylesheet has a url() that it cannot read (line 1).`)
	refuse("escaped url function", "a { background: u\\72l(https://example.com/a.png); }", `The stylesheet spells url( with escape codes (line 1).`)

	// Loading files from other websites
	refuse("import", "@import url(/a.css);", `The stylesheet cannot use @import (line 1).`)
	refuse("import in capitals", `@IMPORT "/a.css";`, `The stylesheet cannot use @import (line 1).`)
	refuse("escaped import", `@\69mport "/a.css";`, `The stylesheet cannot use @import (line 1).`)
	refuse("remote url", "a {\n\tbackground: url(https://example.com/a.png);\n}", `The stylesheet cannot load files from other websites (line 2).`)
	refuse("remote quoted url", `a { background: url("https://example.com/a.png"); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("protocol-relative url", "a { background: url(//example.com/a.png); }", `The stylesheet cannot load files from other websites (line 1).`)
	refuse("backslash for a slash", `a { background: url("/\\example.com/a.png"); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("escaped slashes", `a { background: url(\2F\2F example.com/a.png); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("tab between slashes", "a { background: url(\"/\t/example.com/a.png\"); }", `The stylesheet cannot load files from other websites (line 1).`)
	refuse("scheme in capitals", "a { background: url(HTTPS://example.com/a.png); }", `The stylesheet cannot load files from other websites (line 1).`)
	refuse("scheme without slashes", "a { background: url(https:example.com/a.png); }", `The stylesheet cannot load files from other websites (line 1).`)
	refuse("javascript scheme", `a { background: url("javascript:alert(1)"); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("remote image-set", `a { background: image-set("https://example.com/a.png" 1x); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("remote string nested inside image-set", `a { background: image-set(type("https://example.com/a.png") 1x); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("var fallback inside image-set", `a { background: image-set(var(--u, "https://example.com/a.png") 1x); }`, `The stylesheet cannot use var() inside a function that loads a file (line 1).`)
	refuse("custom property inside image-set", "a {\n\t--x: \"https://example.com/a.png\";\n\tbackground: image-set(var(--x) 1x);\n}", `The stylesheet cannot use var() inside a function that loads a file (line 3).`)
	refuse("escaped var inside image-set", `a { background: image-set(v\61r(--x) 1x); }`, `The stylesheet cannot use var() inside a function that loads a file (line 1).`)
	refuse("env inside cross-fade", `a { background: cross-fade(env(x), "/a.png"); }`, `The stylesheet cannot use env() inside a function that loads a file (line 1).`)
	refuse("attr inside image", `a { background: image(attr(data-x)); }`, `The stylesheet cannot use attr() inside a function that loads a file (line 1).`)
	refuse("redirecting route", `input[value^=a] { background: url(/.intents/Create?account=a@evil.example); }`, `The stylesheet cannot load this website's own pages, only its files (line 1).`)
	refuse("redirecting route, relative", `a { background: url(.intents/Create?account=a@evil.example); }`, `The stylesheet cannot load this website's own pages, only its files (line 1).`)
	refuse("redirecting route inside image-set", `a { background: image-set("/.intents/Create?account=a@evil.example" 1x); }`, `The stylesheet cannot load this website's own pages, only its files (line 1).`)
	refuse("escaped image-set", `a { background: im\61ge-set("https://example.com/a.png" 1x); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("remote src", `a { background: src("https://example.com/a.png"); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("remote image", `a { background: image("https://example.com/a.png"); }`, `The stylesheet cannot load files from other websites (line 1).`)
	refuse("remote font", "@font-face { font-family: X; src: url(https://fonts.example/x.woff2); }", `The stylesheet cannot load files from other websites (line 1).`)

	// Input that the lexer cannot read safely
	refuse("null byte", "a {}\n\x00", `The stylesheet contains a null character (line 2).`)
	refuse("invalid UTF-8", "a::after { content: \"\xff\"; }", `The stylesheet is not valid UTF-8.`)
	refuse("too long", "a{}"+strings.Repeat(" ", maxLength-2), `The stylesheet is longer than 65536 bytes.`)
}

// TestValidate_RemoteAllowed confirms that allowing other websites relaxes only that rule
func TestValidate_RemoteAllowed(t *testing.T) {

	accept := func(name string, value string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.NoError(t, validate(value, true), "validating %q", value)
		})
	}

	refuse := func(name string, value string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.Error(t, validate(value, true), "validating %q", value)
		})
	}

	accept("import", `@import url("https://fonts.example/css?family=X");`)
	accept("remote url", "a { background: url(https://example.com/a.png); }")
	accept("remote image-set", `a { background: image-set("https://example.com/a.png" 1x); }`)
	accept("local url", "a { background: url(/a.png); }")

	refuse("closing tag", "</style>")
	refuse("extra closing brace", "a {} }")
	refuse("unclosed block", "a {")
	refuse("escaped url function", "a { background: u\\72l(https://example.com/a.png); }")
}

// TestValidate_ReportsTheFirstBrokenRule confirms that the error names the earliest problem in the stylesheet
func TestValidate_ReportsTheFirstBrokenRule(t *testing.T) {

	err := Validate("a { background: url(https://example.com/a.png); }\n}\n")
	require.Equal(t, `The stylesheet cannot load files from other websites (line 1).`, derp.Message(err))

	err = Validate("}\na { background: url(https://example.com/a.png); }\n")
	require.Equal(t, `The stylesheet has a '}' with nothing to close (line 1).`, derp.Message(err))
}

// TestValidate_WrappedInScope confirms that an accepted stylesheet stays inside a block that wraps it
func TestValidate_WrappedInScope(t *testing.T) {

	for _, value := range []string{
		"a { color: red; }",
		`a::before { content: "}"; }`,
		"/* } */",
		"@media (width < 600px) { a { color: red; } }",
	} {
		require.NoError(t, Validate(value))
		require.NoError(t, Validate("@scope (#outbox-page) { "+value+" }"), "wrapping %q", value)
		require.Error(t, Validate(value+"}"), "one more closing brace after %q", value)
	}
}

// TestValidate_DoesNotAllocateForEmptyInput pins that an empty stylesheet costs nothing
func TestValidate_DoesNotAllocateForEmptyInput(t *testing.T) {

	if allocs := testing.AllocsPerRun(100, func() { _ = Validate("") }); allocs > 0 {
		t.Errorf("Validate(\"\") allocates %.0f times per call, want 0", allocs)
	}
}
