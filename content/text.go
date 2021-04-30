package content

import (
	"github.com/benpate/html"
	"github.com/benpate/htmlconv"
)

const ItemTypeText = "TEXT"

func TextViewer(lib *Library, b *html.Builder, pm *PathMaker, item *Item) {
	content := item.GetString("text")
	content = htmlconv.FromText(content)
	b.WriteString(content)
}

func TextEditor(lib *Library, b *html.Builder, pm *PathMaker, item *Item) {
	content := item.GetString("text")
	pathID := "body." + pm.NextPath(".")
	b.Container("textarea").ID(pathID).Class("content-editor").InnerHTML(content).Close()
}
