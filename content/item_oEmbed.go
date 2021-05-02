package content

import "github.com/benpate/html"

const ItemTypeOEmbed = "OEMBED"

func OEmbedViewer(library *Library, builder *html.Builder, content Content, id int) {

	item := content[id]

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
			Attr("height", item.GetString("height")).
			Close()
	}
}

func OEmbedEditor(library *Library, builder *html.Builder, content Content, id int) {
	builder.Div().InnerHTML("-- placeholder for oEmbed editor --").Close()
}
