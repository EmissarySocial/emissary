package service

import (
	"io"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/PuerkitoBio/goquery"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/tomnomnom/linkheader"
	"willnorris.com/go/microformats"
)

/*********************
 * Mentions are a W3C standard for connecting conversations across the web.
 *
 * https://indieweb.org/Webmention
 * https://www.w3.org/TR/mention/
 *
 * Golang RegExp syntax:
 * - https://pkg.go.dev/regexp/syntax
 * - https://github.com/google/re2/wiki/Syntax
 *
 *********************/

// Mention defines a service that can send and receive mention data
type Mention struct {
	collection data.Collection
}

// NewMention returns a fully initialized Mention service
func NewMention(collection data.Collection) Mention {
	service := Mention{}
	service.Refresh(collection)
	return service
}

/*******************************************
 * LIFECYCLE METHODS
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Mention) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Mention) Close() {
	// Nothin to do here.
}

/*******************************************
 * COMMON DATA METHODS
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

func (service *Mention) FindLinks(body string) []string {

	result := make([]string, 0)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))

	if err != nil {
		return result
	}

	links := doc.Find("a[href]").Map(getHrefFromNode)

	links = slice.Filter(links, isExternalHref)

	return links
}

// Send will send a mention to the target's endpoint
func (service *Mention) Send(source string, target string) error {

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
func (service *Mention) DiscoverEndpoint(url string) (string, error) {

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

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))

	if err != nil {
		return "", derp.Wrap(err, location, "Error parsing remote document", url)
	}

	linkTag := doc.Find("link[rel=webmention]").First()
	result := linkTag.AttrOr("href", "")

	if result == "" {
		return "", derp.NewBadRequestError(location, "No Mention endpoint found", url)
	}

	return result, nil
}

// Verify confirms that the source document includes a link to the target document
func (service *Mention) Verify(source string, target string, buffer io.Writer) error {

	const location = "service.Mention.Verify"

	var content string

	// Try to load the source document
	txn := remote.Get(source).Response(&content, nil)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error retreiving source", source)
	}

	// Try to parse the source document as HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))

	if err != nil {
		return derp.Wrap(err, location, "Error parsing source", source)
	}

	// Find all anchor tags with an href attribute
	hrefs := doc.Find("a[href]").Map(getHrefFromNode)

	for _, href := range hrefs {

		if href != target {
			continue
		}

		// If buffer exists, write the source document to the buffer, then return without error.
		if buffer != nil {
			buffer.Write([]byte(content))
		}

		return nil
	}

	// Fall through means the link was not found.  Return an error.
	return derp.NewNotFoundError(location, "Target link not found", source, target)
}

func (service *Mention) ParseMicroformats(source io.Reader, sourceURL string) model.Mention {

	mention := model.NewMention()
	mention.SourceURL = sourceURL

	parsedURL, err := url.Parse(sourceURL)

	if err != nil {
		return mention
	}

	// Try to parse microformats in the source document...
	if mf := microformats.Parse(source, parsedURL); mf != nil {
		populateMention(mf, &mention)
	}

	// No errors
	return mention
}

func populateMention(mf *microformats.Data, mention *model.Mention) {

	for _, item := range mf.Items {

		for _, itemType := range item.Type {

			switch itemType {

			// Parse author information [https://microformats.org/wiki/h-card]
			case "h-card":

				if mention.AuthorName == "" {
					mention.AuthorName = convert.String(item.Properties["name"])
				}

				if mention.AuthorName == "" {
					mention.AuthorPhotoURL = convert.String(item.Properties["given-name"])
				}

				if mention.AuthorName == "" {
					mention.AuthorPhotoURL = convert.String(item.Properties["nickname"])
				}

				if mention.AuthorWebsiteURL == "" {
					mention.AuthorWebsiteURL = convert.String(item.Properties["url"])
				}

				if mention.AuthorEmail == "" {
					mention.AuthorEmail = convert.String(item.Properties["email"])
				}

				if mention.AuthorPhotoURL == "" {
					mention.AuthorPhotoURL = convert.String(item.Properties["photo"])
				}

				if mention.AuthorPhotoURL == "" {
					mention.AuthorPhotoURL = convert.String(item.Properties["logo"])
				}

				if mention.AuthorStatus == "" {
					mention.AuthorStatus = convert.String(item.Properties["note"])
				}

				continue

			// Parse entry data
			case "h-entry": // [https://microformats.org/wiki/h-entry]

				if mention.EntryName == "" {
					mention.EntryName = convert.String(item.Properties["name"])
				}

				if mention.EntrySummary == "" {
					mention.EntrySummary = convert.String(item.Properties["summary"])
				}

				if mention.EntryPhotoURL == "" {
					mention.EntryPhotoURL = convert.String(item.Properties["photo"])
				}
			}
		}
	}

	// Last, scan global values for data that may not have been found in the h-entry
	if mention.AuthorWebsiteURL == "" {
		if me, ok := mf.Rels["me"]; ok {
			mention.AuthorWebsiteURL = convert.String(me)
		}
	}
}

// getHrefFromNode returns the [href] value for a given goquery selection
func getHrefFromNode(index int, node *goquery.Selection) string {
	return node.AttrOr("href", "")
}

// isExternalHref returns TRUE if this URL points to an external domain
func isExternalHref(href string) bool {
	return strings.HasPrefix("http://", href) || strings.HasPrefix("https://", href)
}
