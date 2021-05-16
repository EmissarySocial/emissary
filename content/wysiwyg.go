package content

import (
	"strconv"

	"github.com/benpate/html"
)

type WYSIWYG struct{}

func (widget WYSIWYG) View(b *html.Builder, content Content, id int) {
	item := content.GetItem(id)
	result := item.GetString("html")
	b.WriteString(result)
}

func (widget WYSIWYG) Edit(b *html.Builder, content Content, id int, endpoint string) {
	item := content.GetItem(id)
	result := item.GetString("html")
	idString := strconv.Itoa(id)

	b.Form("post", endpoint).Script("install wysiwyg")
	b.Input("hidden", "type").Value("update-item")
	b.Input("hidden", "itemId").Value(idString)
	b.Input("hidden", "check").Value(item.Check)
	b.Input("hidden", "html")
	b.Div().Class("ck-editor").InnerHTML(result)
}
