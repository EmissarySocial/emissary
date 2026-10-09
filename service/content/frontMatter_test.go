package content

import (
	"bytes"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSplitFrontMatter covers the shapes a Markdown file can take at its top
func TestSplitFrontMatter(t *testing.T) {

	// withFrontMatter asserts that a source splits into the expected front matter and body
	withFrontMatter := func(name string, source string, wantMeta mapof.Any, wantBody string) {
		t.Run(name, func(t *testing.T) {
			meta, body, err := splitFrontMatter([]byte(source))
			require.NoError(t, err)
			require.Equal(t, wantMeta, meta)
			require.Equal(t, wantBody, string(body))
		})
	}

	// withoutFrontMatter asserts that a source has no front matter, and is returned whole
	withoutFrontMatter := func(name string, source string, wantBody string) {
		t.Run(name, func(t *testing.T) {
			meta, body, err := splitFrontMatter([]byte(source))
			require.NoError(t, err)
			require.Nil(t, meta)
			require.Equal(t, wantBody, string(body))
		})
	}

	withFrontMatter("basic",
		"---\ntitle: Hello\nrank: 2\n---\n# Body\n",
		mapof.Any{"title": "Hello", "rank": 2}, "# Body\n")

	withFrontMatter("windows line endings",
		"---\r\ntitle: Hello\r\n---\r\n# Body\r\n",
		mapof.Any{"title": "Hello"}, "# Body\r\n")

	withFrontMatter("byte-order mark",
		byteOrderMark+"---\ntitle: Hello\n---\n# Body\n",
		mapof.Any{"title": "Hello"}, "# Body\n")

	withFrontMatter("delimiters with trailing whitespace",
		"--- \t\ntitle: Hello\n---  \n# Body\n",
		mapof.Any{"title": "Hello"}, "# Body\n")

	withFrontMatter("empty block",
		"---\n---\n# Body\n",
		mapof.Any{}, "# Body\n")

	withFrontMatter("block holding only null",
		"---\nnull\n---\n# Body\n",
		mapof.Any{}, "# Body\n")

	withFrontMatter("closing delimiter at end of file",
		"---\ntitle: Hello\n---",
		mapof.Any{"title": "Hello"}, "")

	withFrontMatter("thematic break later in the body is left alone",
		"---\ntitle: Hello\n---\nAbove\n\n---\n\nBelow\n",
		mapof.Any{"title": "Hello"}, "Above\n\n---\n\nBelow\n")

	withFrontMatter("nested values",
		"---\ntags:\n  - one\n  - two\nauthor:\n  name: Ben\n---\n",
		mapof.Any{"tags": []any{"one", "two"}, "author": mapof.Any{"name": "Ben"}}, "")

	withoutFrontMatter("plain markdown", "# Title\n\nBody\n", "# Title\n\nBody\n")
	withoutFrontMatter("empty file", "", "")
	withoutFrontMatter("only a delimiter", "---", "---")
	withoutFrontMatter("unclosed block is a thematic break", "---\nNot front matter\n", "---\nNot front matter\n")
	withoutFrontMatter("four dashes is not a delimiter", "----\ntitle: Hello\n----\n", "----\ntitle: Hello\n----\n")
	withoutFrontMatter("delimiter not on the first line", "\n---\ntitle: Hello\n---\n", "\n---\ntitle: Hello\n---\n")
	withoutFrontMatter("byte-order mark alone", byteOrderMark+"# Title\n", "# Title\n")
}

// TestSplitFrontMatter_Rejects covers front matter that is present but unusable, which must fail
// rather than be published as the body of a page
func TestSplitFrontMatter_Rejects(t *testing.T) {

	// rejects asserts that a source is refused as a bad request
	rejects := func(name string, source string) {
		t.Run(name, func(t *testing.T) {
			meta, body, err := splitFrontMatter([]byte(source))
			require.Error(t, err)
			require.True(t, derp.IsBadRequest(err), "got %v", err)
			require.Nil(t, meta)
			require.Nil(t, body)
		})
	}

	rejects("invalid yaml", "---\ntitle: [unclosed\n---\n# Body\n")
	rejects("a list instead of a mapping", "---\n- one\n- two\n---\n# Body\n")
	rejects("a bare scalar instead of a mapping", "---\njust a string\n---\n# Body\n")
	rejects("block too large", "---\n"+"x: "+strings.Repeat("a", maxFrontMatterBytes)+"\n---\n# Body\n")
}

// TestSplitFrontMatter_MaximumBlock accepts a block of exactly the maximum size
func TestSplitFrontMatter_MaximumBlock(t *testing.T) {

	// "x: " plus the value plus the newline must total exactly maxFrontMatterBytes
	value := strings.Repeat("a", maxFrontMatterBytes-len("x: \n"))
	source := "---\nx: " + value + "\n---\n# Body\n"

	meta, body, err := splitFrontMatter([]byte(source))

	require.NoError(t, err)
	require.Equal(t, value, meta["x"])
	require.Equal(t, "# Body\n", string(body))
}

// FuzzSplitFrontMatter asserts that no input panics, and that a successful split always returns a
// body taken from the end of the input
func FuzzSplitFrontMatter(f *testing.F) {

	f.Add([]byte("---\ntitle: Hello\n---\n# Body\n"))
	f.Add([]byte("---\r\ntitle: Hello\r\n---\r\n"))
	f.Add([]byte(byteOrderMark + "---\n---\n"))
	f.Add([]byte("---\n- a\n---\n"))
	f.Add([]byte("---"))
	f.Add([]byte("---\n"))
	f.Add([]byte("# No front matter"))
	f.Add([]byte("---\n&a [*a, *a]\n---\n"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, source []byte) {

		meta, body, err := splitFrontMatter(source)

		if err != nil {
			return
		}

		trimmed := bytes.TrimPrefix(source, []byte(byteOrderMark))

		// Without front matter, the body is the whole input
		if meta == nil {
			require.Equal(t, trimmed, body)
			return
		}

		// With front matter, the body is whatever followed the closing delimiter
		require.True(t, bytes.HasSuffix(trimmed, body))
	})
}
