package service

import (
	"bytes"
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
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/sherlock"
	"github.com/tomnomnom/linkheader"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mentions are a W3C standard for connecting conversations across the web.
 *
 * https://indieweb.org/Webmention
 * https://www.w3.org/TR/mention/
 *
 * Golang RegExp syntax:
 * - https://pkg.go.dev/regexp/syntax
 * - https://github.com/google/re2/wiki/Syntax
 *
 ******************************************/

// Mention defines a service that can send and receive mention data
type Mention struct {
	factory         *Factory
	ruleService     *Rule
	activityService *ActivityStream
	host            string
}

// NewMention returns a fully initialized Mention service
func NewMention(factory *Factory) Mention {
	return Mention{
		factory: factory,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Mention) Refresh(ruleService *Rule, host string) {
	service.ruleService = ruleService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Mention) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Mention) collection(session data.Session) data.Collection {
	return session.Collection("Mention")
}

// Count returns the number of records that match the provided criteria
func (service *Mention) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice containing all of the Mentions that match the provided criteria
func (service *Mention) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Mention, error) {
	result := make([]model.Mention, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Mentions that match the provided criteria
func (service *Mention) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Mention from the database
func (service *Mention) Load(session data.Session, criteria exp.Expression, mention *model.Mention) error {

	if err := service.collection(session).Load(notDeleted(criteria), mention); err != nil {
		return derp.Wrap(err, "service.Mention.Load", "Error loading Mention", criteria)
	}

	return nil
}

// Save adds/updates an Mention in the database
func (service *Mention) Save(session data.Session, mention *model.Mention, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(mention); err != nil {
		return derp.Wrap(err, "service.Mention.Save", "Error validating Mention", mention)
	}

	// Save the value to the database
	if err := service.collection(session).Save(mention, note); err != nil {
		return derp.Wrap(err, "service.Mention.Save", "Error saving Mention", mention, note)
	}

	return nil
}

// Delete removes an Mention from the database (virtual delete)
func (service *Mention) Delete(session data.Session, mention *model.Mention, note string) error {

	criteria := exp.Equal("_id", mention.MentionID)

	// Delete this Mention
	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Mention.Delete", "Error deleting Mention", criteria)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Mention) ObjectType() string {
	return "Mention"
}

// New returns a fully initialized model.Mention as a data.Object.
func (service *Mention) ObjectNew() data.Object {
	result := model.NewMention()
	return &result
}

func (service *Mention) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Mention); ok {
		return mention.MentionID
	}

	return primitive.NilObjectID
}

func (service *Mention) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Mention) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewMention()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Mention) ObjectSave(session data.Session, object data.Object, comment string) error {
	if mention, ok := object.(*model.Mention); ok {
		return service.Save(session, mention, comment)
	}
	return derp.InternalError("service.Mention.ObjectSave", "Invalid Object Type", object)
}

func (service *Mention) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if mention, ok := object.(*model.Mention); ok {
		return service.Delete(session, mention, comment)
	}
	return derp.InternalError("service.Mention.ObjectDelete", "Invalid Object Type", object)
}

func (service *Mention) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Mention", "Not Authorized")
}

func (service *Mention) Schema() schema.Schema {
	return schema.New(model.MentionSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByOrigin loads an existing Mention by its type/objectID/origin URL
func (service *Mention) LoadByOrigin(session data.Session, objectType string, objectID primitive.ObjectID, originURL string, result *model.Mention) error {

	criteria := exp.Equal("type", objectType).
		AndEqual("objectId", objectID).
		AndEqual("origin.url", originURL)

	return service.Load(session, criteria, result)
}

// LoadOrCreate loads an existing Mention or creates a new one if it doesn't exist
func (service *Mention) LoadOrCreate(session data.Session, objectType string, objectID primitive.ObjectID, originURL string) (model.Mention, error) {

	result := model.NewMention()
	err := service.LoadByOrigin(session, objectType, objectID, originURL, &result)

	// No error means the record was found
	if err == nil {
		return result, nil
	}

	// NotFound error means we should create a new record
	if derp.IsNotFound(err) {
		result.Type = objectType
		result.ObjectID = objectID
		result.Origin.URL = originURL
		return result, nil
	}

	// Other error is bad.  Return the error
	return result, derp.Wrap(err, "service.Mention.LoadOrCreate", "Error loading Mention", objectType, objectID, originURL)
}

func (service *Mention) QueryByObjectID(session data.Session, objectID primitive.ObjectID, options ...option.Option) ([]model.Mention, error) {
	return service.Query(session, exp.Equal("objectId", objectID), options...)
}

/******************************************
 * Web-Mention Helpers
 ******************************************/

// TODO: LOW: This should just use the Locator service.
// ParseURL inspects a target URL and determines what kind of object it is and what the token is.
func (service *Mention) ParseURL(target string) (objectType string, token string, err error) {

	const location = "service.Mention.ParseURL"

	// RULE: If the target URL doesn't start with the service's host, then it
	// doesn't belong on this server
	if !strings.HasPrefix(target, service.host) {
		return "", "", derp.NotFoundError(location, "Target URL is not on this server", target)
	}

	// Parse the URL to ensure that it's valid
	targetURL, err := url.Parse(target)

	if err != nil {
		return "", "", derp.Wrap(err, location, "Error parsing target URL", target)
	}

	// Get the first item in the path.  That's the token we want
	path := list.BySlash(targetURL.Path).Tail()
	token = path.Head()

	// Tokens that begin with "@" are User URLs
	if strings.HasPrefix(token, "@") {
		return model.MentionTypeUser, token[1:], nil
	}

	// Empty tokens reference the Home stream.
	if token == "" {
		return model.MentionTypeStream, "home", nil
	}

	// All other tokens are Stream URLs
	return model.MentionTypeStream, token, nil
}

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
	txn := remote.Get(url).Result(&body)

	if err := txn.Send(); err != nil {
		return "", derp.Wrap(err, location, "Error retrieving remote document", url)
	}

	// Look for Mention link in the response headers
	if value := txn.ResponseHeader().Get("Link"); value != "" {
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
		return "", derp.BadRequestError(location, "No Mention endpoint found", url)
	}

	return result, nil
}

// Verify confirms that the source document includes a link to the target document
func (service *Mention) Verify(source string, target string, buffer io.Writer) error {

	const location = "service.Mention.Verify"

	var content string

	// Try to load the source document
	txn := remote.Get(source).Result(&content)

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
			// nolint:errcheck
			buffer.Write([]byte(content))
		}

		return nil
	}

	// Fall through means the link was not found.  Return an error.
	return derp.NotFoundError(location, "Target link not found", source, target)
}

// TODO: HIGH: This should use a common service to get URL data from Microformats, OpenGraph, JSON-LD, etc.
func (service *Mention) GetPageInfo(body *bytes.Buffer, originURL string, mention *model.Mention) error {

	const location = "service.Mention.GetPageInfo"

	activityService := service.factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)

	// Inspect the source document for metadata (microformats, opengraph, etc.)
	document, err := activityService.Client().Load(originURL, sherlock.AsDocument())

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving page", originURL)
	}

	// Copy the page data into the mention
	mention.Origin.Label = document.Name()
	mention.Origin.URL = document.ID()

	if attributedTo := document.AttributedTo(); attributedTo.NotNil() {
		mention.Author.Name = attributedTo.Name()
		mention.Author.ProfileURL = attributedTo.URL()
		mention.Author.EmailAddress = ""
		mention.Author.IconURL = attributedTo.Icon().URL()
	}

	// No errors
	return nil
}

// getHrefFromNode returns the [href] value for a given goquery selection
func getHrefFromNode(index int, node *goquery.Selection) string {
	return node.AttrOr("href", "")
}

// isExternalHref returns TRUE if this URL points to an external domain
func isExternalHref(href string) bool {
	return strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")
}
