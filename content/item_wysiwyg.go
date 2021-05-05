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
	idString := strconv.Itoa(id)

	b.Form("post", lib.Endpoint).Script("install wysiwygForm")
	{
		b.Input("hidden", "type").Value("update-item")
		b.Input("hidden", "itemId").Value(idString)
		b.Input("hidden", "check").Value(item.Check)
		b.Input("hidden", "html")
		b.Div().Class("ck-editor editor-widget").Script("install wysiwygEditor").InnerHTML(result)
	}

	b.CloseAll()

	b.CloseAll()
}
