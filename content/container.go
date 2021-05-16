package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeContainer = "CONTAINER"

type Container struct{}

func (widget Container) View(builder *html.Builder, content Content, id int) {
	item := content.GetItem(id)

	builder.Div().
		Class("container").
		Data("style", item.GetString("style")).
		Data("size", strconv.Itoa(len(item.Refs)))

	for _, index := range item.Refs {
		builder.Div().Class("container-item")
		content.viewSubTree(builder, index)
		builder.Close()
	}
}

func (widget Container) Edit(builder *html.Builder, content Content, id int, endpoint string) {
	item := content.GetItem(id)

	builder.Div().
		Class("container").
		Data("style", item.GetString("style")).
		Data("size", strconv.Itoa(len(item.Refs))).
		Data("id", strconv.Itoa(id)).
		Data("check", item.Check)

	for _, index := range item.Refs {
		builder.Div().Script("install containerInsertPoint").Close()
		builder.Div().Class("container-item")
		content.editSubTree(builder, index, endpoint)
		builder.Close()
	}
	builder.Div().Script("install containerInsertPoint").Close()
	builder.Close()
}

func (widget Container) Default() Item {
	return Item{}
}
