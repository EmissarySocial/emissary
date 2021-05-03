package content

import (
	"strconv"

	"github.com/benpate/html"
	"github.com/benpate/htmlconv"
)

const ItemTypeText = "TEXT"

func TextViewer(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	result := item.GetString("text")
	result = htmlconv.FromText(result)
	b.WriteString(result)
}

func TextEditor(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	result := item.GetString("text")
	path := "id-" + strconv.Itoa(id)
	b.Container("textarea").ID(path).Class("content-editor").InnerHTML(result)
}
