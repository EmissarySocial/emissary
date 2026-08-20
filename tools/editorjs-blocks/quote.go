package blocks

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
)

// Quote renders an EditorJS "quote" block
type Quote struct{}

// Type returns the EditorJS block name that this renderer handles
func (quote Quote) Type() string {
	return "quote"
}

// GenerateHTML renders the block as a <blockquote> element
func (quote Quote) GenerateHTML(block goeditorjs.EditorJSBlock) (string, error) {

	data := mapof.NewAny()

	if err := json.Unmarshal(block.Data, &data); err != nil {
		return "", derp.Wrap(err, "Reading block data", string(block.Data))
	}

	b := html.New()

	tag := b.Container("blockquote")

	if alignment := data.GetString("alignment"); alignment != "" {
		tag.Attr("style", "text-align: "+alignment)
	}

	tag.InnerHTML(data.GetString("text"))
	tag.Close()

	return b.String(), nil
}
