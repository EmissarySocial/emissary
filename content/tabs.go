package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeTabs = "TABS"

func TabsViewer(lib *Library, builder *html.Builder, content Content, id int) {
	item := content[id]
	labels := item.GetSliceOfString("labels")

	builder.Div().Class("tabs")
	for index, id := range item.Refs {
		nodeID := "#id-" + strconv.Itoa(id)
		label := labels[index]
		builder.A(nodeID).Class("tabs-label").InnerHTML(label).Close()
	}

	for _, id := range item.Refs {
		nodeID := "id-" + strconv.Itoa(id)
		builder.Div().ID(nodeID).EndBracket()
		lib.SubTree(builder, content, id)
		builder.Close()
	}

	builder.Close()
}

func TabsEditor(lib *Library, builder *html.Builder, content Content, id int) {
	item := content[id]
	labels := item.GetSliceOfString("labels")

	builder.Div().Class("tabs")
	for index, id := range item.Refs {
		nodeID := "#id-" + strconv.Itoa(id)
		label := labels[index]
		builder.A(nodeID).Class("tabs-label").InnerHTML(label).Close()
	}

	for _, id := range item.Refs {
		nodeID := "id-" + strconv.Itoa(id)
		builder.Div().ID(nodeID).EndBracket()
		lib.SubTree(builder, content, id)
		builder.Close()
	}

	builder.Close()
}
