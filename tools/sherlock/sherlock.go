/*
Package sherlock is a library for extracting metadata from web pages.
It uses as many methods as possible to extract page data, including:
- Open Graph
- Microformats2

Coming Soon..
- HTML Meta Tags
- oEmbed
- JSON-LD
- Twitter Cards?
*/
package sherlock

import (
	"bytes"
	"net/url"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

func Load(target string) (Page, error) {

	const location = "sherlock.Load"

	// Load the document
	var body bytes.Buffer

	// Load the document from the URL
	transaction := remote.Get(target).Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return Page{}, derp.Wrap(err, location, "Error loading URL", target)
	}

	// Parse and return
	return Parse(target, &body)
}

func Parse(target string, body *bytes.Buffer) (Page, error) {

	const location = "sherlock.Parse"

	// Validate the URL
	parsedURL, err := url.Parse(target)

	if err != nil {
		return Page{}, derp.Wrap(err, location, "Error parsing URL", target)
	}

	result := NewPage()
	bodyBytes := body.Bytes()

	// Try OpenGraph (via HTMLInfo)
	ParseOpenGraph(target, bytes.NewReader(bodyBytes), &result)

	// Try Microformats2
	ParseMicroFormats(parsedURL, bytes.NewReader(bodyBytes), &result)

	// Look for JSON-LD embedded in the docuemnt
	ParseJSONLD(body, &result)

	// If we don't have a canonical URL, then use the target URL
	if result.CanonicalURL == "" {
		result.CanonicalURL = target
	}

	// If we have SOMETHING to work with, then call it here.
	if !result.IsEmpty() {
		return result, nil
	}

	// Otherwise, look links to an external JSON-LD result
	if ok := ParseLinkedJSONLD(body, &result); ok {
		return result, nil
	}

	// Otherwise, look for an oEmbed provider for this document
	if ok := ParseOEmbed(bytes.NewReader(bodyBytes), &result); ok {
		return result, nil
	}

	return result, nil
}
