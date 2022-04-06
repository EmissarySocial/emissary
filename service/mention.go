package service

import (
	"regexp"
	"strings"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/remote"
	"github.com/tomnomnom/linkheader"
	"github.com/whisperverse/whisperverse/model"
)

/*********************
 * Mentions are a W3C standard for connection conversations across the web.
 *
 * https://indieweb.org/Webmention
 * https://www.w3.org/TR/mention/
 *
 * Golang RegExp syntax:
 * - https://pkg.go.dev/regexp/syntax
 * - https://github.com/google/re2/wiki/Syntax
 *
 *********************/

var mentionSendTag *regexp.Regexp
var mentionSendAttr *regexp.Regexp
var mentionVerifyTag *regexp.Regexp

func init() {
	mentionSendTag = regexp.MustCompile(`(?i)<[a|link][^>]+?rel=["']?mention["']?[^>]*?>`)
	mentionSendAttr = regexp.MustCompile(`(?i)href=['"]?([^\t\n\v\f\r "'>]+)['"]?`)
	mentionVerifyTag = regexp.MustCompile(`(?i)<(a|img|video)[^>]+?(href|src)=["']?([^\t\n\v\f\r "'>]+)["']?[^>]*>`) // \S => NOT WHITESPACE
}

// Mention defines a service that can send and receive mention data
type Mention struct {
	collection data.Collection
}

// NewMention returns a fully initialized Mention service
func NewMention(collection data.Collection) Mention {
	return Mention{
		collection: collection,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns an iterator containing all of the Mentions who match the provided criteria
func (service *Mention) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Mention from the database
func (service *Mention) Load(criteria exp.Expression, stream *model.Mention) error {

	if err := service.collection.Load(notDeleted(criteria), stream); err != nil {
		return derp.Wrap(err, "service.Mention.Load", "Error loading Mention", criteria)
	}

	return nil
}

// Save adds/updates an Mention in the database
func (service *Mention) Save(stream *model.Mention, note string) error {

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, "service.Mention.Save", "Error saving Mention", stream, note)
	}

	return nil
}

// Delete removes an Mention from the database (virtual delete)
func (service *Mention) Delete(stream *model.Mention, note string) error {

	criteria := exp.Equal("_id", stream.StreamID)

	// Delete this Mention
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Mention.Delete", "Error deleting Mention", criteria)
	}

	return nil
}

/*******************************************
 * WEB-MENTION HELPERS
 *******************************************/

// Send will send a mention to the target's endpoint
func (service Mention) Send(source string, target string) error {

	const location = "service.Mention.Sent"

	// Try to look up the target's endpoint
	endpoint, err := service.DiscoverEndpoint(target)

	if err != nil {
		return derp.Wrap(err, location, "Error discovering endpoint", source, target)
	}

	// Try to send the mention data to the endpoint
	txn := remote.Post(endpoint).
		Form("source", source).
		Form("target", target)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending mention", source, target)
	}

	// Silence means success
	return nil
}

// DiscoverEndpoint tries to find the Mention endpoint for the provided URL
func (service Mention) DiscoverEndpoint(url string) (string, error) {

	const location = "service.Mention.Discover"
	var body string

	txn := remote.Get(url).Response(&body, nil)

	if err := txn.Send(); err != nil {
		return "", derp.Wrap(err, location, "Error retrieving remote document", url)
	}

	// Look for Mention link in the response headers
	if value := txn.ResponseObject.Header.Get("Link"); value != "" {
		links := linkheader.Parse(value)

		for _, link := range links {
			if strings.ToLower(link.Rel) == "mention" {
				return link.URL, nil
			}
		}
	}

	// Look for Mention links in the response body
	if tag := mentionSendTag.FindString(body); tag != "" {
		attributes := mentionSendAttr.FindStringSubmatch(tag)
		if len(attributes) == 2 {
			return attributes[1], nil
		}
	}

	return "", derp.NewNotFoundError(location, "Webmention link not found in URL", url)
}

// Verify confirms that the source document includes a link to the target document
func (service Mention) Verify(source string, target string) error {

	const location = "service.Mention.Verify"

	var content string

	// Try to load the source document
	txn := remote.Get(source).Response(&content, nil)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error retreiving source", source)
	}

	// Scan for links to the target URL
	links := mentionVerifyTag.FindAllStringSubmatch(content, -1)

	for _, link := range links {
		if link[3] == target {
			return nil // If found, success.  No error returned.
		}
	}

	return derp.NewNotFoundError(location, "Target link not found", source, target)
}
