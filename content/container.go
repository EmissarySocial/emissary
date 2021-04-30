package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeContainer = "CONTAINER"

func ContainerViewer(library *Library, builder *html.Builder, pm *PathMaker, item *Item) {
	builder.Div().Class("container container-" + item.GetString("style") + " container-size-" + strconv.Itoa(len(item.Kids)))
	for index := range item.Kids {
		builder.Div().EndBracket()
		library.SubTree(builder, pm, &item.Kids[index])
		builder.Close()
	}
	builder.Close()
}

func ContainerEditor(library *Library, builder *html.Builder, pm *PathMaker, item *Item) {
	builder.Div().Class("container container-" + item.GetString("style") + " container-size-" + strconv.Itoa(len(item.Kids)))
	for index := range item.Kids {
		builder.Div().EndBracket()
		library.SubTree(builder, pm, &item.Kids[index])
		builder.Close()
	}
	builder.Close()
}
