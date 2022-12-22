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
	"github.com/davecgh/go-spew/spew"
	"github.com/tomnomnom/linkheader"
)

// discoverLinks attempts to discover ActivityPub/RSS/Atom/JSONFeed links from a given following URL.
func discoverLinks(targetURL string) []digit.Link {

	const location = "service.discoverLinks"

	// Look for links embedded in the HTML
	if result := discoverLinksFromHTML(targetURL); len(result) > 0 {
		return result
	}

	// Fall back to WebFinger, just in case
	if result := discoverLinksFromWebFinger(targetURL); len(result) > 0 {
		return result
	}

	// Fall through, fail through
	derp.Report(derp.NewBadRequestError(location, "Unable to discover links from WebFinger or HTML", targetURL))
	return make([]digit.Link, 0)
}

func discoverLinksFromHTML(targetURL string) []digit.Link {

	const location = "service.discoverLinksFromHTML"

	var body bytes.Buffer

	result := make([]digit.Link, 0)

	// Try to load the targetURL.
	transaction := remote.Get(targetURL).Response(&body, nil)

	if err := transaction.Send(); err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading URL", targetURL))
		return result
	}

	spew.Dump(transaction.ResponseObject.Header)

	// Scan the response headers for WebSub links
	// TODO: LOW: Are RSS links ever put in the headers also?
	linkHeaders := linkheader.ParseMultiple(transaction.ResponseObject.Header["Link"])

	for _, link := range linkHeaders {

		switch link.Rel {
		case model.LinkRelationHub:
			result = append(result, digit.Link{
				MediaType:    model.MagicMimeTypeWebSub,
				RelationType: link.Rel,
				Href:         link.URL,
			})

		case model.LinkRelationSelf:
			result = append(result, digit.Link{
				RelationType: link.Rel,
				Href:         link.URL,
			})

		}
	}

	// If the document itself is an RSS feed, then we're done.  Add it to the list.
	// TODO: LOW: Possibly parse RSS-Cloud here?
	mimeType := transaction.ResponseObject.Header.Get("Content-Type")
	mimeType = list.Semicolon(mimeType).First()

	switch mimeType {
	case model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML:
		return append(result, digit.Link{
			RelationType: model.LinkRelationSelf,
			MediaType:    mimeType,
			Href:         targetURL,
		})
	}

	// Fall through assumes that this is an HTML document.
	// So, look for embedded links to other feeds (ActivityPub/RSS/Atom/JSONFeed).

	// Scan the HTML document for relevant links
	htmlDocument, err := goquery.NewDocumentFromReader(&body)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error parsing HTML document"))
		return result
	}

	links := htmlDocument.Find("link[rel=alternate],link[rel=self],link[rel=hub],atom:link").Nodes

	// Look through RSS links for all valid feeds
	for _, link := range links {

		relationType := nodeAttribute(link, "rel")
		href := nodeAttribute(link, "href")
		href = getRelativeURL(targetURL, href)

		// Special case for WebSub relation types
		switch relationType {
		case model.LinkRelationHub:

			result = append(result, digit.Link{
				RelationType: model.LinkRelationHub,
				MediaType:    model.MagicMimeTypeWebSub,
				Href:         href,
			})
			continue
		}

		// General case for all other relation types
		mediaType := nodeAttribute(link, "type")
		mediaType = list.Semicolon(mediaType).First()

		switch mediaType {

		case model.MimeTypeActivityPub, model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS:
			result = append(result, digit.Link{
				RelationType: model.LinkRelationAlternate,
				MediaType:    mediaType,
				Href:         href,
			})
		}
	}

	return result
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
	// TODO: LOW: Look into Textcasting? http://textcasting.org

	return result, derp.NewNotFoundError(location, "Error parsing following URL", targetURL)
}
