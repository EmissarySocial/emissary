package content

import (
	"strconv"

	"github.com/benpate/html"
)

type HTML struct{}

func (widget HTML) View(b *html.Builder, content Content, id int) {
	item := content.GetItem(id)
	result := item.GetString("html")
	b.WriteString(result)
}

func (widget HTML) Edit(b *html.Builder, content Content, id int, endpoint string) {
	item := content.GetItem(id)
	result := item.GetString("html")
	idString := strconv.Itoa(id)

	b.Form("post", endpoint).Script("install wysiwyg")
	b.Input("hidden", "type").Value("update-item")
	b.Input("hidden", "itemId").Value(idString)
	b.Input("hidden", "check").Value(item.Check)
	b.Container("textarea").Name("html").InnerHTML(result)
}
