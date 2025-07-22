package blocks

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
)

type Code struct{}

func (code Code) Type() string {
	return "code"
}

func (code Code) GenerateHTML(block goeditorjs.EditorJSBlock) (string, error) {

	data := mapof.NewAny()

	if err := json.Unmarshal(block.Data, &data); err != nil {
		return "", derp.Wrap(err, "Unable to read block data", string(block.Data))
	}

	b := html.New()
	b.Container("pre")
	tag := b.Container("code")

	tag.InnerText(data.GetString("code"))
	b.CloseAll()

	return b.String(), nil
}
