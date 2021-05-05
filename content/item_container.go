package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeContainer = "CONTAINER"

func ContainerViewer(library *Library, builder *html.Builder, content Content, id int) {
	item := content[id]

	builder.Div().
		Class("container").
		Data("style", item.GetString("style")).
		Data("size", strconv.Itoa(len(item.Refs)))

	for _, index := range item.Refs {
		builder.Div().Class("container-item")
		library.SubTree(builder, content, index)
		builder.Close()
	}
}

func ContainerEditor(library *Library, builder *html.Builder, content Content, id int) {
	item := content[id]

	builder.Div().
		Class("container").
		Data("style", item.GetString("style")).
		Data("size", strconv.Itoa(len(item.Refs))).
		Data("id", strconv.Itoa(id)).
		Data("check", item.Check)

	for _, index := range item.Refs {
		builder.Div().Script("install containerInsertPoint").Close()
		builder.Div().Class("container-item")
		library.SubTree(builder, content, index)
		builder.Close()
	}
	builder.Div().Script("install containerInsertPoint").Close()
	builder.Close()
}
