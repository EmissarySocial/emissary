package content

import (
	"github.com/benpate/html"
	"github.com/davecgh/go-spew/spew"
)

const ItemTypeWYSIWYG = "WYSIWYG"

func WYSIWYGViewer(lib *Library, b *html.Builder, pm *PathMaker, item *Item) {
	content := item.GetString("html")
	spew.Dump(content)
	b.WriteString(content)
}

func WYSIWYGEditor(lib *Library, b *html.Builder, pm *PathMaker, item *Item) {
	content := item.GetString("html")
	pathID := "body." + pm.NextPath(".")
	b.Div().ID(pathID).Class("ck-editor").InnerHTML(content).Close()
}
