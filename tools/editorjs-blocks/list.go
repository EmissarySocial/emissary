package blocks

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
)

type List struct{}

func (list List) Type() string {
	return "list"
}

func (list List) GenerateHTML(block goeditorjs.EditorJSBlock) (string, error) {

	data := mapof.NewAny()

	if err := json.Unmarshal(block.Data, &data); err != nil {
		return "", derp.Wrap(err, "Unable to read block data", string(block.Data))
	}

	b := html.New()
	return generateList(b, data.GetString("style"), data.GetSliceOfMap("items")), nil
}

func generateList(b *html.Builder, style string, items []mapof.Any) string {

	var list *html.Element

	switch style {

	case "ordered":
		list = b.Container("ol")
	default:
		list = b.Container("ul")
	}

	for _, item := range items {
		li := b.Container("li")
		li.InnerText(item.GetString("content"))

		if children := item.GetSliceOfMap("items"); len(children) > 0 {
			generateList(b.SubTree(), style, children)
		}
		li.Close()
	}

	list.Close()

	return b.String()
}
