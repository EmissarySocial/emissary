package content

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/markdown"
	"github.com/stretchr/testify/require"
)

// testSourceURL is a raw file on a forge, the kind of address a StreamSource stores
const testSourceURL = "https://raw.example.com/owner/repo/main/docs/guide.md"

// urls returns the addresses from a list of AttachmentRefs, for compact assertions
func urls(refs []AttachmentRef) []string {

	result := make([]string, 0, len(refs))

	for _, ref := range refs {
		result = append(result, ref.URL)
	}

	return result
}

// TestFindAttachments_Markdown confirms that links written in Markdown are found once the file
// is rendered, which is the only form the sync ever reads
func TestFindAttachments_Markdown(t *testing.T) {

	source := "![Flow](attachments/flow.png)\n\n[Manual](attachments/manual.pdf)\n\n![Demo](attachments/demo.mp4)\n"
	refs := FindAttachments(markdown.ToHTML(source), testSourceURL)

	require.Equal(t, []AttachmentRef{
		{URL: "https://raw.example.com/owner/repo/main/docs/attachments/flow.png", Kind: model.AttachmentMediaTypeImage},
		{URL: "https://raw.example.com/owner/repo/main/docs/attachments/manual.pdf", Kind: model.AttachmentMediaTypeDocument},
		{URL: "https://raw.example.com/owner/repo/main/docs/attachments/demo.mp4", Kind: model.AttachmentMediaTypeVideo},
	}, refs)
}

// TestFindAttachments_CodeBlocksAreText confirms that a path mentioned inside a code sample is
// never imported.  This is why links are read from the rendered HTML and not from the source.
func TestFindAttachments_CodeBlocksAreText(t *testing.T) {

	source := "Use `![x](attachments/inline.png)` like this:\n\n```\n<img src=\"attachments/fenced.png\">\n```\n"
	refs := FindAttachments(markdown.ToHTML(source), testSourceURL)

	require.Empty(t, refs)
}

// TestFindAttachments_Rule pins which links count as attachments
func TestFindAttachments_Rule(t *testing.T) {

	tests := map[string]struct {
		link     string
		expected string // "" when the link must NOT be imported
	}{
		"beside the file":           {"attachments/a.png", "https://raw.example.com/owner/repo/main/docs/attachments/a.png"},
		"shared folder above":       {"../attachments/a.png", "https://raw.example.com/owner/repo/main/attachments/a.png"},
		"nested inside the folder":  {"attachments/screens/a.png", "https://raw.example.com/owner/repo/main/docs/attachments/screens/a.png"},
		"absolute, same host":       {"https://raw.example.com/other/repo/main/attachments/a.png", "https://raw.example.com/other/repo/main/attachments/a.png"},
		"host differs only in case": {"https://RAW.example.com/owner/repo/main/attachments/a.png", "https://RAW.example.com/owner/repo/main/attachments/a.png"},
		"fragment is dropped":       {"attachments/a.png#top", "https://raw.example.com/owner/repo/main/docs/attachments/a.png"},
		"uppercase extension":       {"attachments/A.PNG", "https://raw.example.com/owner/repo/main/docs/attachments/A.PNG"},
		"another host":              {"https://cdn.example.com/attachments/a.png", ""},
		"another scheme":            {"http://raw.example.com/owner/repo/main/docs/attachments/a.png", ""},
		"no attachments folder":     {"images/a.png", ""},
		"a file NAMED attachments":  {"docs/attachments", ""},
		"folder name is exact":      {"my-attachments/a.png", ""},
		"extension not allowed":     {"attachments/a.zip", ""},
		"svg is never imported":     {"attachments/a.svg", ""},
		"another markdown file":     {"attachments/next.md", ""},
		"no extension":              {"attachments/readme", ""},
		"unparseable":               {"attachments/%zz.png", ""},
		"too long to store":         {"attachments/" + strings.Repeat("a", 2048) + ".png", ""},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			refs := FindAttachments(`<p><img src="`+test.link+`"></p>`, testSourceURL)

			if test.expected == "" {
				require.Empty(t, refs)
				return
			}

			require.Equal(t, []string{test.expected}, urls(refs))
		})
	}
}

// TestFindAttachments_QueryIsDropped records what happens to a cgit address, whose branch lives
// in the query string: a relative link resolves without it, so the file comes from the default branch
func TestFindAttachments_QueryIsDropped(t *testing.T) {

	refs := FindAttachments(`<img src="attachments/a.png">`, "https://git.example.com/repo.git/plain/README.md?h=master")
	require.Equal(t, []string{"https://git.example.com/repo.git/plain/attachments/a.png"}, urls(refs))
}

// TestFindAttachments_Duplicates confirms that a file linked twice is imported once, in the
// position of its first appearance
func TestFindAttachments_Duplicates(t *testing.T) {

	document := `<a href="attachments/b.png"><img src="attachments/b.png"></a><img src="attachments/a.png"><img src="attachments/b.png#again">`
	refs := FindAttachments(document, testSourceURL)

	require.Equal(t, []string{
		"https://raw.example.com/owner/repo/main/docs/attachments/b.png",
		"https://raw.example.com/owner/repo/main/docs/attachments/a.png",
	}, urls(refs))
}

// TestFindAttachments_OnlyLinks confirms that attributes other than img[src] and a[href] are ignored
func TestFindAttachments_OnlyLinks(t *testing.T) {

	document := `<iframe src="attachments/a.png"></iframe><img alt="attachments/b.png"><a title="attachments/c.png">x</a><div src="attachments/d.png"></div>`
	require.Empty(t, FindAttachments(document, testSourceURL))
}

// TestFindAttachments_BadSource confirms that a source address that cannot be parsed imports nothing
func TestFindAttachments_BadSource(t *testing.T) {
	require.Empty(t, FindAttachments(`<img src="attachments/a.png">`, "https://bad host/%zz"))
}

// TestRewriteAttachments confirms that imported links are replaced and nothing else changes
func TestRewriteAttachments(t *testing.T) {

	document := "<p>Before &amp; after</p>\n<p><img src=\"attachments/flow.png\" alt=\"Flow &quot;chart&quot;\"></p>\n<p><a href=\"attachments/manual.pdf\" rel=\"nofollow\">Manual</a></p>\n<pre><code>attachments/flow.png</code></pre>\n<img src=\"https://cdn.example.com/x.png\">"

	localURLs := map[string]string{
		"https://raw.example.com/owner/repo/main/docs/attachments/flow.png":   "https://site.example/s1/attachments/a1",
		"https://raw.example.com/owner/repo/main/docs/attachments/manual.pdf": "https://site.example/s1/attachments/a2",
	}

	result := RewriteAttachments(document, testSourceURL, localURLs)

	require.Equal(t, "<p>Before &amp; after</p>\n<p><img src=\"https://site.example/s1/attachments/a1\" alt=\"Flow &#34;chart&#34;\"></p>\n<p><a href=\"https://site.example/s1/attachments/a2\" rel=\"nofollow\">Manual</a></p>\n<pre><code>attachments/flow.png</code></pre>\n<img src=\"https://cdn.example.com/x.png\">", result)
}

// TestRewriteAttachments_NotImportedStaysRemote confirms that an attachment link with no local
// copy (beyond the cap, or not yet imported) is written back unchanged
func TestRewriteAttachments_NotImportedStaysRemote(t *testing.T) {

	document := `<img src="attachments/flow.png">`
	require.Equal(t, document, RewriteAttachments(document, testSourceURL, map[string]string{}))
}

// TestRewriteAttachments_MediaPlayers confirms that an image link to a video or audio file
// becomes a player, and that a plain link to one stays a link
func TestRewriteAttachments_MediaPlayers(t *testing.T) {

	source := "![Demo <1>](attachments/demo.mp4)\n\n![Theme](attachments/theme.mp3)\n\n[Download](attachments/demo.mp4)\n"
	document := markdown.ToHTML(source)

	localURLs := map[string]string{
		"https://raw.example.com/owner/repo/main/docs/attachments/demo.mp4":  "https://site.example/s1/attachments/v1",
		"https://raw.example.com/owner/repo/main/docs/attachments/theme.mp3": "https://site.example/s1/attachments/a1",
	}

	result := RewriteAttachments(document, testSourceURL, localURLs)

	require.Contains(t, result, `<video controls src="https://site.example/s1/attachments/v1">Demo &lt;1&gt;</video>`)
	require.Contains(t, result, `<audio controls src="https://site.example/s1/attachments/a1">Theme</audio>`)
	require.Contains(t, result, `<a href="https://site.example/s1/attachments/v1" rel="nofollow">Download</a>`)
	require.NotContains(t, result, "<img")
}

// TestRewriteAttachments_BadSource confirms that a source that cannot be parsed changes nothing
func TestRewriteAttachments_BadSource(t *testing.T) {
	document := `<img src="attachments/a.png">`
	require.Equal(t, document, RewriteAttachments(document, "https://bad host/%zz", map[string]string{"x": "y"}))
}

// TestAttachmentKind pins the extension table, and that the path alone is read
func TestAttachmentKind(t *testing.T) {

	require.Equal(t, model.AttachmentMediaTypeImage, AttachmentKind("https://x.example/attachments/a.webp?raw=true"))
	require.Equal(t, model.AttachmentMediaTypeVideo, AttachmentKind("attachments/a.MOV"))
	require.Equal(t, model.AttachmentMediaTypeAudio, AttachmentKind("attachments/a.flac"))
	require.Equal(t, model.AttachmentMediaTypeDocument, AttachmentKind("attachments/a.pdf"))
	require.Equal(t, "", AttachmentKind("attachments/a.svg"))
	require.Equal(t, "", AttachmentKind("https://x.example/a.png%zz"))
}

// TestMaxAttachmentBytes pins the caps in D7 of GIT-MARKDOWN-STREAM-ATTACHMENTS.md
func TestMaxAttachmentBytes(t *testing.T) {
	require.Equal(t, int64(10<<20), MaxAttachmentBytes(model.AttachmentMediaTypeImage))
	require.Equal(t, int64(10<<20), MaxAttachmentBytes(model.AttachmentMediaTypeDocument))
	require.Equal(t, int64(100<<20), MaxAttachmentBytes(model.AttachmentMediaTypeVideo))
	require.Equal(t, int64(100<<20), MaxAttachmentBytes(model.AttachmentMediaTypeAudio))
}
