package service

import (
	"bytes"
	"mime"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/PuerkitoBio/goquery"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/rosetta/list"
	"github.com/kr/jsonfeed"
	"github.com/tomnomnom/linkheader"
)

// discoverLinks attempts to discover ActivityPub/RSS/Atom/JSONFeed links from a given following URL.
func discoverLinks(response *http.Response, body *bytes.Buffer) digit.LinkSet {

	result := digit.NewLinkSet(10)

	// Look for links embedded in the HTTP headers
	discoverLinks_Headers(&result, response)

	// Look for links embedded in the HTML
	// nolint:errcheck // derp.Report is good enough here.
	if err := discoverLinks_HTML(&result, response, body); err != nil {
		derp.Report(derp.Wrap(err, "service.discoverLinks", "Error getting links from HTML"))
	}

	// Fall back to WebFinger, just in case
	if len(result) == 0 {
		discoverLinks_WebFinger(&result, response.Request.URL.String())
	}

	// Return all results
	return result
}

// discoverLinks_Headers scans the HTTP headers for WebSub links
func discoverLinks_Headers(result *digit.LinkSet, response *http.Response) {

	if response == nil {
		return
	}

	// Scan the response headers for WebSub links
	// TODO: LOW: Are RSS links ever put into the headers?
	// TODO: LOW: Are RSSCloud links ever put into the headers?
	linkHeaders := linkheader.ParseMultiple(response.Header["Link"])

	for _, link := range linkHeaders {

		switch link.Rel {

		case model.LinkRelationHub:
			result.Append(digit.Link{
				MediaType:    model.MagicMimeTypeWebSub,
				RelationType: link.Rel,
				Href:         link.URL,
			})

		default:
			result.Append(digit.Link{
				RelationType: link.Rel,
				Href:         link.URL,
			})
		}
	}
}

func discoverLinks_HTML(result *digit.LinkSet, response *http.Response, body *bytes.Buffer) error {

	const location = "service.discoverLinks_HTML"

	// If the document itself is an RSS feed, then we're done.  Add it to the list.
	mimeType := response.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(mimeType)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing media type", mimeType)
	}

	switch mediaType {
	case
		model.MimeTypeJSONFeed,
		model.MimeTypeAtom,
		model.MimeTypeRSS,
		model.MimeTypeXML,
		model.MimeTypeXMLText:

		// TODO: LOW: Possibly parse RSS-Cloud here?

		result.Apply(digit.Link{
			RelationType: model.LinkRelationSelf,
			MediaType:    mediaType,
			Href:         response.Request.URL.String(),
		})
	}

	// Fall through assumes that this is an HTML document.
	// So, look for embedded links to other feeds (ActivityPub/RSS/Atom/JSONFeed).

	// Scan the HTML document for relevant links
	htmlDocument, err := goquery.NewDocumentFromReader(bytes.NewReader(body.Bytes()))

	if err != nil {
		return derp.Wrap(err, location, "Error parsing HTML document")
	}

	links := htmlDocument.Find("[rel=alternate],[rel=self],[rel=hub],[rel=icon],[rel=feed]").Nodes

	// Look through RSS links for all valid feeds
	for _, link := range links {

		relationType := nodeAttribute(link, "rel")
		href := nodeAttribute(link, "href")
		href = getRelativeURL(response.Request.URL.String(), href)

		// Special case for WebSub relation types
		switch relationType {
		case model.LinkRelationIcon:

			result.Apply(digit.Link{
				RelationType: model.LinkRelationIcon,
				Href:         href,
			})
			continue

		case model.LinkRelationHub:

			result.Apply(digit.Link{
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

		case
			model.MimeTypeActivityPub,
			model.MimeTypeJSONFeed,
			model.MimeTypeAtom,
			model.MimeTypeRSS,
			model.MimeTypeXML,
			model.MimeTypeXMLText:

			result.Apply(digit.Link{
				RelationType: relationType,
				MediaType:    mediaType,
				Href:         href,
			})
		}
	}

	return nil
}

// discoverLinks_WebFinger uses the WebFinger protocol to search for additional metadata about the targetURL
func discoverLinks_WebFinger(result *digit.LinkSet, targetURL string) {

	// Send a GET request to the WebFinger service
	resource, err := digit.Lookup(targetURL)

	if err != nil {
		derp.Report(err)
		return
	}

	for _, link := range resource.Links {
		result.Append(link)
	}
}

func discoverLinks_RSS(response *http.Response, body *bytes.Buffer) []digit.Link {

	result := make(digit.LinkSet, 0)

	discoverLinks_Headers(&result, response)

	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body.Bytes()))

	if err != nil {
		derp.Report(derp.Wrap(err, "service.discoverLinks_RSS", "Error parsing RSS document"))
		return result
	}

	links := document.Find("[rel=hub],[rel=self]").Nodes

	for _, link := range links {

		// Hacky way to skip over non-link nodes because the query library won't do it for us.
		if (link.Data != "link") && (link.Data != "atom:link") {
			continue
		}

		relation := nodeAttribute(link, "rel")
		switch relation {
		case
			model.LinkRelationHub,
			model.LinkRelationSelf:

			href := nodeAttribute(link, "href")
			mimeType := nodeAttribute(link, "type")
			link := digit.NewLink(relation, mimeType, href)
			result = append(result, link)
		}
	}

	return result
}

func discoverLinks_JSONFeed(response *http.Response, jsonFeed *jsonfeed.Feed) []digit.Link {

	result := make(digit.LinkSet, 0)
	discoverLinks_Headers(&result, response)

	// Discover hubs
	for _, hub := range jsonFeed.Hubs {

		switch strings.ToUpper(hub.Type) {

		case model.FollowMethodWebSub:
			result = append(result, digit.NewLink(model.LinkRelationHub, model.MimeTypeJSONFeed, hub.URL))
		}
	}

	return result
}
