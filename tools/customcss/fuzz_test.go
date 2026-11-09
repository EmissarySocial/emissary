package customcss

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzValidate confirms that Validate never panics, and that whatever it accepts is safe to embed
func FuzzValidate(f *testing.F) {

	for _, seed := range []string{
		"",
		"a { color: red; }",
		"nav > a::before { content: \"}\"; }",
		"@media (width < 600px) { a { color: red; } }",
		"a { background: url(/a.png); }",
		"a { background: url(https://example.com/a.png); }",
		"a { background: image-set(\"/a.png\" 1x); }",
		"a { background: u\\72l(//x); }",
		"@\\69mport \"x\";",
		"</style><script>alert(1)</script>",
		"a {} }",
		"a { \"unterminated",
		"/* unterminated",
		"url(\"/a\\\")",
		"\x00",
		"\xff",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {

		remoteErr := validate(value, true)

		// Allowing other websites only ever accepts more
		if strictErr := Validate(value); (strictErr == nil) && (remoteErr != nil) {
			t.Fatalf("accepted %q strictly, but refused it with other websites allowed: %v", value, remoteErr)
		}

		if remoteErr != nil {
			return
		}

		// An accepted stylesheet cannot close a <style> element, and the lexer read all of it
		if strings.Contains(value, "</") {
			t.Fatalf("accepted %q, which contains \"</\"", value)
		}

		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
			t.Fatalf("accepted %q, which the lexer cannot read to the end", value)
		}

		// An accepted stylesheet leaves nothing open, so it stays inside a wrapper and cannot close one
		if err := validate("@scope (#p) { "+value+" }", true); err != nil {
			t.Fatalf("accepted %q, but not wrapped in @scope: %v", value, err)
		}

		if validate(value+"}", true) == nil {
			t.Fatalf("accepted %q, and also accepted it with one more '}'", value)
		}

		// ...and it cannot swallow the first character the page writes after it
		if validate(value+"}}", true) == nil {
			t.Fatalf("accepted %q, and also accepted it with two more '}'", value)
		}
	})
}
