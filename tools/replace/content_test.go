package replace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContent_Naive(t *testing.T) {
	original := "This is an example of a string to replace"
	result := Content(original, "is", "was")
	require.Equal(t, "Thwas was an example of a string to replace", result)
}

func TestContent_AtBeginning(t *testing.T) {
	original := "#Here's a tag at the beginning"
	result := Content(original, "#Here", "<b>#Here</b>")
	require.Equal(t, "<b>#Here</b>'s a tag at the beginning", result)
}

func TestContent_AtEnd(t *testing.T) {
	original := "Here's a tag at the #end"
	result := Content(original, "#end", "<b>#end</b>")
	require.Equal(t, "Here's a tag at the <b>#end</b>", result)
}

func TestContent_SkipHTML(t *testing.T) {
	original := "Here's some <a href='server.com/#tag'>HTML</a> with a #tag somewhere"
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, "Here's some <a href='server.com/#tag'>HTML</a> with a <b>#tag</b> somewhere", result)
}

func TestContent_SkipHTML_AtEnd(t *testing.T) {
	original := "Here's some <a href='server.com/#tag'>HTML</a> with a #tag at the #end"
	result := Content(original, "#end", "<b>#end</b>")
	require.Equal(t, "Here's some <a href='server.com/#tag'>HTML</a> with a #tag at the <b>#end</b>", result)
}

func TestContent_CaseInsensitive(t *testing.T) {
	original := "THIS this ThIs tHiS is case insensitive"
	result := Content(original, "this", "THAT")
	require.Equal(t, "THAT THAT THAT THAT is case insensitive", result)
}

// TestContent_SkipAnchorBody confirms that a #tag inside the text of an <a> element is left alone.
func TestContent_SkipAnchorBody(t *testing.T) {
	original := `A link <a href="/search?q=%23tag">#tag</a> stays put`
	result := Content(original, "#tag", `<a href="/search?q=%23tag">#tag</a>`)
	require.Equal(t, original, result)
}

// TestContent_TagAfterAnchor confirms that a #tag AFTER a closing </a> is still replaced.
func TestContent_TagAfterAnchor(t *testing.T) {
	original := `<a href="/x">#tag</a> and another #tag`
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, `<a href="/x">#tag</a> and another <b>#tag</b>`, result)
}

// TestContent_UpperCaseAnchor confirms that anchor detection is case-insensitive.
func TestContent_UpperCaseAnchor(t *testing.T) {
	original := `<A HREF="/x">#tag</A> then #tag`
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, `<A HREF="/x">#tag</A> then <b>#tag</b>`, result)
}

// TestContent_ArticleIsNotAnchor confirms that <article> (a tag starting with "a") is not mistaken for an anchor.
func TestContent_ArticleIsNotAnchor(t *testing.T) {
	original := `<article>#tag</article>`
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, `<article><b>#tag</b></article>`, result)
}

// TestContent_NestedTagsInsideAnchor confirms that child tags inside an anchor do not end the anchor body early.
func TestContent_NestedTagsInsideAnchor(t *testing.T) {
	original := `<a href="/x"><b>#tag</b></a> and #tag`
	result := Content(original, "#tag", "<i>#tag</i>")
	require.Equal(t, `<a href="/x"><b>#tag</b></a> and <i>#tag</i>`, result)
}

// TestContent_BareAnchor confirms that a bare <a> (no attributes) still protects its body.
func TestContent_BareAnchor(t *testing.T) {
	original := `<a>#tag</a> #tag`
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, `<a>#tag</a> <b>#tag</b>`, result)
}

// TestContent_UnterminatedAnchor confirms that an <a> with no closing tag protects the rest of the string.
func TestContent_UnterminatedAnchor(t *testing.T) {
	original := `text <a href="/x">#tag and #more`
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, original, result)
}

// TestContent_AnchorAtEOF confirms that a truncated "<a" at the end of the string does not panic or mis-detect.
func TestContent_AnchorAtEOF(t *testing.T) {
	original := "trailing <a"
	result := Content(original, "#tag", "<b>#tag</b>")
	require.Equal(t, original, result)
}

// TestContent_Idempotent confirms that linkifying the same content twice yields identical output (no nesting).
func TestContent_Idempotent(t *testing.T) {
	original := "Testing hashtags #travel here"
	replacement := `<a href="/search?q=%23travel">#travel</a>`

	once := Content(original, "#travel", replacement)
	twice := Content(once, "#travel", replacement)

	require.Equal(t, `Testing hashtags <a href="/search?q=%23travel">#travel</a> here`, once)
	require.Equal(t, once, twice, "linkification must be idempotent")
}

// TestContent_EmptyMatch confirms that an empty match string leaves the original untouched (no panic, no loop).
func TestContent_EmptyMatch(t *testing.T) {
	require.Equal(t, "abc", Content("abc", "", "X"))
}

// TestContent_EmptyOriginal confirms that an empty original returns empty.
func TestContent_EmptyOriginal(t *testing.T) {
	require.Equal(t, "", Content("", "#tag", "<b>#tag</b>"))
}

// FuzzContent_Idempotent asserts that wrapping a match in an anchor is idempotent for any input.
func FuzzContent_Idempotent(f *testing.F) {

	const match = "#tag"
	const replacement = `<a href="/s">#tag</a>`

	f.Add("Testing #tag here")
	f.Add(`<a href="/s">#tag</a>`)
	f.Add("#tag</a>#tag")
	f.Add("<a>#tag")
	f.Add("<article>#tag</article>")
	f.Add("")

	f.Fuzz(func(t *testing.T, original string) {
		once := Content(original, match, replacement)
		twice := Content(once, match, replacement)
		require.Equal(t, once, twice, "linkification must be idempotent for %q", original)
	})
}
