package content

import (
	"regexp"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestNewMarkdownItem splits a file into its parts and hashes the whole of it
func TestNewMarkdownItem(t *testing.T) {

	file := []byte("---\ntitle: Hello\n---\n# Body\n")

	item, err := NewItem(model.ContentFormatMarkdown, file)

	require.NoError(t, err)
	require.Equal(t, model.ContentFormatMarkdown, item.Format)
	require.Equal(t, "# Body\n", string(item.Source))
	require.Equal(t, "Hello", item.Meta["title"])
	require.Equal(t, hashContent(file), item.Hash)
}

// TestNewMarkdownItem_HashCoversFrontMatter changes the hash when only the front matter changes, so a
// retitled page is still seen as changed
func TestNewMarkdownItem_HashCoversFrontMatter(t *testing.T) {

	before, err := NewItem(model.ContentFormatMarkdown, []byte("---\ntitle: Before\n---\n# Body\n"))
	require.NoError(t, err)

	after, err := NewItem(model.ContentFormatMarkdown, []byte("---\ntitle: After\n---\n# Body\n"))
	require.NoError(t, err)

	require.Equal(t, before.Source, after.Source)
	require.NotEqual(t, before.Hash, after.Hash)
}

// TestNewMarkdownItem_InvalidFrontMatter reports front matter that cannot be read
func TestNewMarkdownItem_InvalidFrontMatter(t *testing.T) {

	item, err := NewItem(model.ContentFormatMarkdown, []byte("---\ntitle: [unclosed\n---\n"))

	require.True(t, derp.IsBadRequest(err))
	require.Equal(t, Item{}, item)
}

// TestHashContent returns sixteen lowercase hexadecimal characters, the same every time
func TestHashContent(t *testing.T) {

	hash := hashContent([]byte("Emissary"))

	require.Regexp(t, regexp.MustCompile(`^[0-9a-f]{16}$`), hash)
	require.Equal(t, hash, hashContent([]byte("Emissary")))
	require.NotEqual(t, hash, hashContent([]byte("emissary")))
	require.Regexp(t, regexp.MustCompile(`^[0-9a-f]{16}$`), hashContent(nil))
}
