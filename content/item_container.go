package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeContainer = "CONTAINER"

func ContainerViewer(library *Library, builder *html.Builder, content Content, id int) {
	item := content[id]
	builder.Div().Class("container container-" + item.GetString("style") + " container-size-" + strconv.Itoa(len(item.Refs)))
	for _, index := range item.Refs {
		builder.Div().EndBracket()
		library.SubTree(builder, content, index)
		builder.Close()
	}
	builder.Close()
}

func ContainerEditor(library *Library, builder *html.Builder, content Content, id int) {
	item := content[id]
	builder.Div().Class("container container-" + item.GetString("style") + " container-size-" + strconv.Itoa(len(item.Refs)))
	for _, index := range item.Refs {
		builder.Div().EndBracket()
		library.SubTree(builder, content, index)
		builder.Close()
	}
	builder.Close()
}
