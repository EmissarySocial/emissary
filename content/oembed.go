package content

import "github.com/benpate/html"

type OEmbed struct{}

func (widget OEmbed) View(builder *html.Builder, content Content, id int) {

	item := content.GetItem(id)

	// If the oEmbed data includes HTML, then just use that and be done.
	if html := item.GetString("html"); html != "" {
		builder.WriteString(html)
		return
	}

	// Special handling for known types
	switch item.GetString("type") {

	case "photo":
		builder.Empty("img").
			Attr("src", item.GetString("url")).
			Attr("width", item.GetString("width")).
			Attr("height", item.GetString("height"))
	}
}

func (widget OEmbed) Edit(builder *html.Builder, content Content, id int, endpoint string) {
	builder.Div().InnerHTML("-- placeholder for oEmbed editor --")
}
