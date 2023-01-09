package as

import (
	"net/url"

	"github.com/go-fed/activity/streams"
)

func SetLink(item HasSetLink, hrefProperty *url.URL, nameProperty string, typeProperty string) {

	link := streams.NewActivityStreamsLink()

	// Set HREF
	href := streams.NewActivityStreamsHrefProperty()
	href.SetIRI(hrefProperty)
	link.SetActivityStreamsHref(href)

	// Set NAME
	if nameProperty != "" {
		name := streams.NewActivityStreamsNameProperty()
		name.AppendXMLSchemaString(nameProperty)
	}

	// Set TYPE
	if typeProperty != "" {
		mediaType := streams.NewActivityStreamsMediaTypeProperty()
		mediaType.Set(typeProperty)
		link.SetActivityStreamsMediaType(mediaType)
	}

	item.SetActivityStreamsLink(link)
}

func SetURL(item HasSetURL, value *url.URL) {
	property := streams.NewActivityStreamsUrlProperty()
	property.AppendIRI(value)
	item.SetActivityStreamsUrl(property)
}
