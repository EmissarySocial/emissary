package content

import (
	"strconv"

	"github.com/benpate/html"
	"github.com/benpate/htmlconv"
)

type Text struct{}

func (widget Text) View(b *html.Builder, content Content, id int) {
	item := content.GetItem(id)
	result := item.GetString("text")
	result = htmlconv.FromText(result)
	b.WriteString(result)
}

func (widget Text) Edit(b *html.Builder, content Content, id int, endpoint string) {
	item := content.GetItem(id)
	result := item.GetString("text")
	idString := strconv.Itoa(id)

	b.Form("post", endpoint).Script("install wysiwyg")
	b.Input("hidden", "type").Value("update-item")
	b.Input("hidden", "itemId").Value(idString)
	b.Input("hidden", "check").Value(item.Check)
	b.Container("textarea").Name("text").InnerHTML(result)
}
