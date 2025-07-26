package blocks

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"github.com/davidscottmills/goeditorjs"
)

type Table struct{}

func (table Table) Type() string {
	return "table"
}

func (table Table) GenerateHTML(block goeditorjs.EditorJSBlock) (string, error) {
	spew.Dump("table -----", block)

	data := mapof.NewAny()

	if err := json.Unmarshal(block.Data, &data); err != nil {
		return "", derp.Wrap(err, "Unable to read block data", string(block.Data))
	}

	spew.Dump(data)

	return "", nil
}
