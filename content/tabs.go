package content

import (
	"github.com/benpate/convert"
	"github.com/benpate/html"
)

const ItemTypeTabs = "TABS"

func TabsViewer(lib *Library, builder *html.Builder, pm *PathMaker, item *Item) {
	labels := convert.SliceOfString(item.GetInterface("labels"))

	builder.Div().Class("tabs")
	for _, label := range labels {
		id := "#tab-" + pm.NextPath("-")
		builder.A(id).Class("tabs-label").InnerHTML(label).Close()
	}

	pm.Rewind(len(labels))

	for index := range item.Kids {
		id := "tab-" + pm.NextPath("-")
		builder.Div().ID(id).EndBracket()
		lib.SubTree(builder, pm, &item.Kids[index])
		builder.Close()
	}

	builder.Close()
}

func TabsEditor(lib *Library, builder *html.Builder, pm *PathMaker, item *Item) {

	labels := convert.SliceOfString(item.GetInterface("labels"))

	builder.Div().Class("tabs")
	for _, label := range labels {
		id := "#tab-" + pm.NextPath("-")
		builder.A(id).Class("tabs-label").InnerHTML(label).Close()
	}

	pm.Rewind(len(labels))

	for index := range item.Kids {
		id := "tab-" + pm.NextPath("-")
		builder.Div().ID(id).EndBracket()
		lib.SubTree(builder, pm, &item.Kids[index])
		builder.Close()
	}

	builder.Close()
}
