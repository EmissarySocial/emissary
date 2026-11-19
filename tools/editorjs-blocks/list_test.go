package blocks

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidscottmills/goeditorjs"
	"github.com/stretchr/testify/require"
)

// listBlock wraps raw block data in the EditorJS envelope that GenerateHTML receives
func listBlock(data string) goeditorjs.EditorJSBlock {
	return goeditorjs.EditorJSBlock{Type: "list", Data: json.RawMessage(data)}
}

// renderList renders one list block's data, failing the test on an error
func renderList(t *testing.T, data string) string {
	t.Helper()

	result, err := List{}.GenerateHTML(listBlock(data))
	require.NoError(t, err, "rendering %s", data)
	return result
}

// TestList_Type confirms the EditorJS block name this renderer registers for
func TestList_Type(t *testing.T) {
	require.Equal(t, "list", List{}.Type())
}

// TestList_Unordered confirms that the default style is a bulleted list, in item order
func TestList_Unordered(t *testing.T) {
	result := renderList(t, `{"style":"unordered","items":[{"content":"one","items":[]},{"content":"two","items":[]}]}`)
	require.Equal(t, `<ul><li>one</li><li>two</li></ul>`, result)
}

// TestList_Ordered confirms that the ordered style is a numbered list
func TestList_Ordered(t *testing.T) {
	result := renderList(t, `{"style":"ordered","items":[{"content":"one","items":[]}]}`)
	require.Equal(t, `<ol><li>one</li></ol>`, result)
}

// TestList_OtherStyles confirms that any style but "ordered", including none, is a bulleted list
func TestList_OtherStyles(t *testing.T) {
	require.Equal(t, `<ul><li>one</li></ul>`, renderList(t, `{"style":"checklist","items":[{"content":"one"}]}`))
	require.Equal(t, `<ul><li>one</li></ul>`, renderList(t, `{"items":[{"content":"one"}]}`))
}

// TestList_InlineMarkup confirms that an item's inline HTML is rendered, not shown as text
func TestList_InlineMarkup(t *testing.T) {

	// Content.Format sanitizes the whole document afterwards, so the renderer escapes nothing
	item := `<b>bold</b>, <i>italic</i>, a <a href=\"https://example.com/\">link</a>,<br>and a <span class=\"cdx-text-color\" style=\"color: #0055cc;\">color</span>`
	result := renderList(t, `{"style":"unordered","items":[{"content":"`+item+`","items":[]}]}`)

	require.Equal(t, `<ul><li><b>bold</b>, <i>italic</i>, a <a href="https://example.com/">link</a>,<br>and a <span class="cdx-text-color" style="color: #0055cc;">color</span></li></ul>`, result)
	require.NotContains(t, result, "&lt;")
}

// TestList_Nested confirms that child items render as a list inside their parent's item
func TestList_Nested(t *testing.T) {
	result := renderList(t, `{"style":"ordered","items":[{"content":"parent","items":[{"content":"child","items":[{"content":"grandchild","items":[]}]}]},{"content":"sibling","items":[]}]}`)
	require.Equal(t, `<ol><li>parent<ol><li>child<ol><li>grandchild</li></ol></li></ol></li><li>sibling</li></ol>`, result)
}

// TestList_Empty confirms that a list with no items, or no items key at all, is an empty list
func TestList_Empty(t *testing.T) {
	require.Equal(t, `<ul></ul>`, renderList(t, `{"style":"unordered","items":[]}`))
	require.Equal(t, `<ul></ul>`, renderList(t, `{}`))
}

// TestList_MissingContent confirms that an item without content is an empty item
func TestList_MissingContent(t *testing.T) {
	require.Equal(t, `<ul><li></li></ul>`, renderList(t, `{"items":[{"items":[]}]}`))
}

// TestList_LegacyStringItems pins how List 1.x data renders, whose items are plain strings
func TestList_LegacyStringItems(t *testing.T) {

	// A string carries no content key, so each is an empty item until the editor saves the block again
	require.Equal(t, `<ul><li></li><li></li></ul>`, renderList(t, `{"style":"unordered","items":["one","two"]}`))
}

// TestList_MalformedData confirms that block data which is not a JSON object is an error
func TestList_MalformedData(t *testing.T) {

	for _, data := range []string{``, `not json`, `[1,2,3]`, `"text"`, `{"items":`} {
		result, err := List{}.GenerateHTML(listBlock(data))
		require.Error(t, err, "data %q", data)
		require.Empty(t, result, "data %q", data)
	}
}

// FuzzList_GenerateHTML confirms that no block data panics, and that every success is one whole list
func FuzzList_GenerateHTML(f *testing.F) {

	f.Add(`{"style":"unordered","items":[{"content":"one","items":[]}]}`)
	f.Add(`{"style":"ordered","items":[{"content":"<b>x</b>","items":[{"content":"y","items":[]}]}]}`)
	f.Add(`{"items":["legacy"]}`)
	f.Add(`{"items":[{"content":5,"items":"nope"}]}`)
	f.Add(`{"items":null}`)
	f.Add(`{}`)
	f.Add(`not json`)
	f.Add("\xff")

	f.Fuzz(func(t *testing.T, data string) {

		result, err := List{}.GenerateHTML(listBlock(data))

		if err != nil {
			require.Empty(t, result)
			return
		}

		isUnordered := strings.HasPrefix(result, "<ul>") && strings.HasSuffix(result, "</ul>")
		isOrdered := strings.HasPrefix(result, "<ol>") && strings.HasSuffix(result, "</ol>")
		require.True(t, isUnordered || isOrdered, "not one whole list: %q", result)
	})
}
