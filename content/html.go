package content

import (
	"github.com/benpate/html"
)

const ItemTypeHTML = "HTML"

func HTMLViewer(lib *Library, b *html.Builder, pm *PathMaker, item *Item) {
	content := item.GetString("html")
	b.WriteString(content)
}

func HTMLEditor(lib *Library, b *html.Builder, pm *PathMaker, item *Item) {
	content := item.GetString("html")
	pathID := "body." + pm.NextPath(".")
	b.Container("textarea").ID(pathID).Class("content-editor").InnerHTML(content).Close()
}
