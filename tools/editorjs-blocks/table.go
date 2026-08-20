package blocks

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
)

// Table renders an EditorJS "table" block
type Table struct{}

// Type returns the EditorJS block name that this renderer handles
func (table Table) Type() string {
	return "table"
}

// GenerateHTML validates the block data, but does not yet emit any markup
func (table Table) GenerateHTML(block goeditorjs.EditorJSBlock) (string, error) {

	data := mapof.NewAny()

	if err := json.Unmarshal(block.Data, &data); err != nil {
		return "", derp.Wrap(err, "Reading block data", string(block.Data))
	}

	return "", nil
}
