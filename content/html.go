package content

import (
	"strconv"

	"github.com/benpate/html"
)

const ItemTypeHTML = "HTML"

func HTMLViewer(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	result := item.GetString("html")
	b.WriteString(result)
}

func HTMLEditor(lib *Library, b *html.Builder, content Content, id int) {
	item := content[id]
	nodeID := "id-" + strconv.Itoa(id)
	result := item.GetString("html")
	b.Container("textarea").ID(nodeID).Class("html-editor").InnerHTML(result).Close()
}
