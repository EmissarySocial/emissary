package customcss

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUnescape confirms that each kind of CSS escape decodes the way a browser decodes it
func TestUnescape(t *testing.T) {

	check := func(name string, value string, want string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			require.Equal(t, want, unescape(value), "unescaping %q", value)
		})
	}

	check("no escapes", "image-set", "image-set")
	check("empty", "", "")
	check("hex escape", `u\72l`, "url")
	check("hex escape in capitals", `\2F\2F`, "//")
	check("hex escape in lowercase", `\2f\2f`, "//")
	check("one space ends a hex escape", `\2F x`, "/x")
	check("only one space belongs to the escape", `\2F  x`, "/ x")
	check("tab ends a hex escape", "\\2F\tx", "/x")
	check("newline ends a hex escape", "\\2F\nx", "/x")
	check("CRLF ends a hex escape", "\\2F\r\nx", "/x")
	check("six digits at most", `\0000411`, "A1")
	check("NUL becomes U+FFFD", `\0`, "\uFFFD")
	check("surrogate becomes U+FFFD", `\D800`, "\uFFFD")
	check("past the last code point becomes U+FFFD", `\110000`, "\uFFFD")
	check("last code point", `\10FFFF`, "\U0010FFFF")
	check("escaped character", `\:`, ":")
	check("escaped backslash", `\\`, `\`)
	check("escaped multi-byte character", `\é`, "é")
	check("line continuation", "a\\\nb", "ab")
	check("CRLF line continuation", "a\\\r\nb", "ab")
	check("form feed line continuation", "a\\\fb", "ab")
	check("carriage return line continuation", "a\\\rb", "ab")
	check("trailing backslash", `a\`, "a")
}

// TestIdentifier confirms that names are unescaped and lowercased before they are compared
func TestIdentifier(t *testing.T) {
	require.Equal(t, "import", identifier([]byte(`\69MPORT`)))
	require.Equal(t, "image-set", identifier([]byte(`Image-Set`)))
	require.Equal(t, "", identifier(nil))
}

// TestHexValue confirms every hex digit's value
func TestHexValue(t *testing.T) {

	for index, digit := range "0123456789abcdef" {
		require.Equal(t, index, hexValue(byte(digit)))
	}

	for index, digit := range "ABCDEF" {
		require.Equal(t, index+10, hexValue(byte(digit)))
	}
}

// TestIsTerminatedString confirms that only a string closed by an unescaped quote counts as closed
func TestIsTerminatedString(t *testing.T) {

	check := func(value string, want bool) {
		t.Helper()
		t.Run(value, func(t *testing.T) {
			require.Equal(t, want, isTerminatedString([]byte(value)), "reading %q", value)
		})
	}

	check(`""`, true)
	check(`''`, true)
	check(`"abc"`, true)
	check(`"a\\"`, true)
	check(`"\22"`, true)
	check(`"a\ "`, true)
	check(`"`, false)
	check(``, false)
	check(`"abc`, false)
	check(`"abc'`, false)
	check(`"\"`, false)
	check(`"a\\\"`, false)
}

// TestIsTerminatedComment confirms that a comment must end with its own "*/"
func TestIsTerminatedComment(t *testing.T) {
	require.True(t, isTerminatedComment([]byte("/**/")))
	require.True(t, isTerminatedComment([]byte("/* x */")))
	require.False(t, isTerminatedComment([]byte("/*/")))
	require.False(t, isTerminatedComment([]byte("/* x")))
	require.False(t, isTerminatedComment(nil))
}

// TestURLValue confirms that the address comes out of every spelling of url()
func TestURLValue(t *testing.T) {

	check := func(name string, value string, want string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			got, isTerminated := urlValue([]byte(value))
			require.True(t, isTerminated, "reading %q", value)
			require.Equal(t, want, got, "reading %q", value)
		})
	}

	checkOpen := func(name string, value string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			_, isTerminated := urlValue([]byte(value))
			require.False(t, isTerminated, "reading %q", value)
		})
	}

	check("unquoted", "url(/a.png)", "/a.png")
	check("double-quoted", `url("/a.png")`, "/a.png")
	check("single-quoted", `url('/a.png')`, "/a.png")
	check("padded", "url(  /a.png\n)", "/a.png")
	check("padded and quoted", `url( "/a.png" )`, "/a.png")
	check("empty", "url()", "")
	check("empty quotes", `url("")`, "")
	check("escaped", `url(\2F\2F x)`, "//x")
	check("escaped inside quotes", `url("\2F\2F x")`, "//x")
	check("capitals", "URL(/a.png)", "/a.png")

	checkOpen("no closing parenthesis", "url(/a.png")
	checkOpen("quote never closes", `url("/a.png)`)
	checkOpen("escaped quote", `url("/a.png\")`)
	checkOpen("no opening parenthesis", "url")
	checkOpen("escaped closing parenthesis", `url(\)`)
	checkOpen("escaped closing parenthesis after an address", `url(/a.png\)`)

	check("escaped backslash before the closing parenthesis", `url(/a\\)`, `/a\`)
}
