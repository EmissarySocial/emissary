package content

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.yaml.in/yaml/v3"
)

// splitFrontMatter separates a block of YAML front matter from the Markdown that follows it.
// It returns nil front matter when the source has none.
func splitFrontMatter(source []byte) (mapof.Any, []byte, error) {

	const location = "content.splitFrontMatter"

	// A byte-order mark would otherwise hide the opening delimiter
	source = bytes.TrimPrefix(source, []byte(byteOrderMark))

	// RULE: Front matter exists only when the very first line opens it
	firstLine, remainder, _ := bytes.Cut(source, []byte("\n"))

	if !isFrontMatterDelimiter(firstLine) {
		return nil, source, nil
	}

	// RULE: A block that never closes is not front matter.  It is a thematic break.
	block, body, isClosed := cutFrontMatterBlock(remainder)

	if !isClosed {
		return nil, source, nil
	}

	// RULE: A block this large is not a list of page settings
	if len(block) > maxFrontMatterBytes {
		return nil, nil, derp.BadRequest(location, "Front matter is too large", len(block))
	}

	// Parse the block as a YAML mapping
	meta := mapof.NewAny()

	if err := yaml.Unmarshal(block, &meta); err != nil {
		return nil, nil, derp.Wrap(err, location, "Front matter is not valid YAML", derp.WithBadRequest())
	}

	// A block holding only `null` decodes to a nil map, which would read as "no front matter"
	if meta == nil {
		meta = mapof.NewAny()
	}

	return meta, body, nil
}

// cutFrontMatterBlock finds the line that closes a block of front matter, and returns the block
// before that line and the body after it
func cutFrontMatterBlock(value []byte) ([]byte, []byte, bool) {

	// Nothing after the opening delimiter means there is nothing to close it
	if len(value) == 0 {
		return nil, nil, false
	}

	for offset := 0; ; {

		line, remainder, hasNewline := bytes.Cut(value[offset:], []byte("\n"))

		if isFrontMatterDelimiter(line) {
			return value[:offset], remainder, true
		}

		// The last line has been read, and it did not close the block
		if !hasNewline {
			return nil, nil, false
		}

		offset = len(value) - len(remainder)
	}
}

// isFrontMatterDelimiter returns TRUE if a line opens or closes a block of front matter
func isFrontMatterDelimiter(line []byte) bool {
	return string(bytes.TrimRight(line, " \t\r")) == frontMatterDelimiter
}
