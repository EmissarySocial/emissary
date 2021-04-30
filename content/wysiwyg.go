package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeWYSIWYG = "WYSIWYG"

func WYSIWYGViewer(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	result := item.GetString("html")
	b.WriteString(result)
}

func WYSIWYGEditor(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	result := item.GetString("html")
	path := "id-" + strconv.Itoa(id)

	b.Div().ID(path).Class("ck-editor").InnerHTML(result).Close()
}
