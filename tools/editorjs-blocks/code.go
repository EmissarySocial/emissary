package blocks

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
)

// Code renders an EditorJS "code" block
type Code struct{}

// Type returns the EditorJS block name that this renderer handles
func (code Code) Type() string {
	return "code"
}

// GenerateHTML renders the block as a <pre><code> element
func (code Code) GenerateHTML(block goeditorjs.EditorJSBlock) (string, error) {

	data := mapof.NewAny()

	if err := json.Unmarshal(block.Data, &data); err != nil {
		return "", derp.Wrap(err, "Reading block data", string(block.Data))
	}

	b := html.New()
	b.Container("pre")
	tag := b.Container("code")

	tag.InnerText(data.GetString("code"))
	b.CloseAll()

	return b.String(), nil
}
