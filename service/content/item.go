package content

import (
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/cespare/xxhash/v2"
)

// Item is one piece of content, as a remote source offered it
type Item struct {
	Format string    // Content format of Source: MARKDOWN or HTML, as contentFormat decided
	Source []byte    // Raw content with any front matter removed.  Never pre-rendered HTML.
	Meta   mapof.Any // Front matter values, or nil when the source carries no front matter
	Hash   string    // Hexadecimal hash of the complete source, front matter included
}

// NewItem splits a file into its front matter and its body, in the format its source declared
func NewItem(format string, file []byte) (Item, error) {

	const location = "content.NewItem"

	meta, body, err := splitFrontMatter(file)

	if err != nil {
		return Item{}, derp.Wrap(err, location, "Reading front matter")
	}

	// The hash covers the front matter, so that a change to a title alone is still a change
	return Item{
		Format: format,
		Source: body,
		Meta:   meta,
		Hash:   hashContent(file),
	}, nil
}

// hashContent returns the hexadecimal xxhash of a byte slice
func hashContent(value []byte) string {
	return fmt.Sprintf("%016x", xxhash.Sum64(value))
}
