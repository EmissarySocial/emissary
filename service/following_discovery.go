package service

import (
	"bytes"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/PuerkitoBio/goquery"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/list"
)

// discoverLinks attempts to discover ActivityPub/RSS/Atom/JSONFeed links from a given following URL.
func discoverLinks(targetURL string) ([]digit.Link, error) {

	const location = "service.discoverLinks"

	// Next, look for links embedded in the HTML
	if result, err := discoverLinksFromHTML(targetURL); err != nil {
		return nil, derp.Wrap(err, location, "Error discovering links from HTML", targetURL)
	} else if len(result) > 0 {
		return sortLinks(result), nil
	}

	// Try to use WebFinger first
	if result := discoverLinksFromWebFinger(targetURL); len(result) > 0 {
		return sortLinks(result), nil
	}

	// Fall through, fail through
	return make([]digit.Link, 0), derp.NewBadRequestError(location, "Unable to discover links from WebFinger or HTML", targetURL)
}

func discoverLinksFromWebFinger(targetURL string) []digit.Link {

	// Compute the WebFinger service for the targetURL
	webfingerURL, err := getWebFingerURL(targetURL)

	if err != nil {
		return make([]digit.Link, 0)
	}

	// Send a GET request to the WebFinger service
	object := digit.NewResource("")
	transaction := remote.Get(webfingerURL.String()).Response(&object, nil)

	if err := transaction.Send(); err != nil {
		return make([]digit.Link, 0)
	}

	if object.Links == nil {
		return make([]digit.Link, 0)
	}

	return object.Links
}

func discoverLinksFromHTML(targetURL string) ([]digit.Link, error) {

	const location = "service.discoverLinksFromHTML"

	var body bytes.Buffer

	result := make([]digit.Link, 0)

	// Try to load the
	transaction := remote.Get(targetURL).Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Error loading HTML document", targetURL)
	}

	mimeType := transaction.ResponseObject.Header.Get("Content-Type")
	mimeType = list.Semicolon(mimeType).First()

	// If the document itself is an RSS feed, then success.
	switch mimeType {
	case model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML:
		return []digit.Link{{
			RelationType: "alternate",
			MediaType:    mimeType,
			Href:         targetURL,
		}}, nil
	}

	// Otherwise, try to parse the document as an HTML file with embedded links to the ActivityPub/RSS/Atom/JSONFeed feeds.
	htmlDocument, err := goquery.NewDocumentFromReader(&body)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, location, "Error parsing HTML document"))
	}

	links := htmlDocument.Find("link[rel=alternate],link[rel=self]").Nodes

	// Look through RSS links for all valid feeds
	for _, link := range links {

		mediaType := nodeAttribute(link, "type")
		mediaType = list.Semicolon(mediaType).First()

		switch mediaType {

		case model.MimeTypeActivityPub, model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS:
			result = append(result, digit.Link{
				RelationType: "alternate",
				MediaType:    mediaType,
				Href:         getRelativeURL(targetURL, nodeAttribute(link, "href")),
			})
		}
	}

	return result, nil
}

func getWebFingerURL(targetURL string) (url.URL, error) {

	const location = "service.getWebFingerURL"
	var result url.URL

	// Try to parse the followingURL as a standard URL
	if parsedURL, err := url.Parse(targetURL); err == nil {

		result.Scheme = parsedURL.Scheme
		result.Host = parsedURL.Host
		result.Path = "/.well-known/webfinger"
		result.RawQuery = "resource=" + targetURL

		return result, nil
	}

	// TODO: HIGH: Try to parse as a Mastodon username @benpate@mastodon.social
	// TODO: MEDIUM: Try to parse as an email address??

	return result, derp.NewNotFoundError(location, "Error parsing following URL", targetURL)
}

func sortLinks(links []digit.Link) []digit.Link {

	result := make([]digit.Link, 0, len(links))
	mediaTypes := []string{model.MimeTypeActivityPub, model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML}

	for _, mediaType := range mediaTypes {
		for _, link := range links {
			if link.MediaType == mediaType {
				result = append(result, link)
			}
		}
	}

	return result
}
