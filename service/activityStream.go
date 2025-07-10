package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ascacherules"
	"github.com/EmissarySocial/emissary/tools/ascontextmaker"
	"github.com/EmissarySocial/emissary/tools/ashash"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"go.mongodb.org/mongo-driver/mongo"
)

// ActivityStream implements the Hannibal HTTP client interface, and provides a cache for ActivityStream documents.
type ActivityStream struct {
	domainService       *Domain
	locatorService      *Locator
	searchDomainService *SearchDomain
	searchQueryService  *SearchQuery
	streamService       *Stream
	userService         *User
	collection          *mongo.Collection
	innerClient         streams.Client
	cacheClient         *ascache.Client
	hostname            string
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewActivityStream creates a new ActivityStream service
func NewActivityStream() ActivityStream {
	return ActivityStream{}
}

// Refresh updates the ActivityStream service with new dependencies
func (service *ActivityStream) Refresh(collection *mongo.Collection, domainService *Domain, locatorService *Locator, searchDomainService *SearchDomain, searchQueryService *SearchQuery, streamService *Stream, userService *User, hostname string) {
	service.domainService = domainService
	service.locatorService = locatorService
	service.searchDomainService = searchDomainService
	service.searchQueryService = searchQueryService
	service.streamService = streamService
	service.userService = userService
	service.collection = collection
	service.hostname = hostname
	service.innerClient = nil
	service.cacheClient = nil
}

func (service *ActivityStream) initClients() {

	// Build a new client stack
	sherlockClient := sherlock.NewClient(
		sherlock.WithUserAgent(service.hostname + " (emissary.social)"),
	)

	// Try to attach a private key to this client
	if privateKey, err := service.domainService.PrivateKey(); err == nil {
		publicKeyID := service.domainService.PublicKeyID()
		sherlockClient.WithOptions(
			sherlock.WithActor(publicKeyID, privateKey),
		)

	} else {
		derp.Report(derp.Wrap(err, "service.ActivityStream.client", "Error loading private key"))
	}

	// enforce opinionated data formats
	normalizerClient := asnormalizer.New(sherlockClient)

	// compute document context (if missing)
	contextMakerClient := ascontextmaker.New(normalizerClient)

	// apply custom caching rules to documents
	cacheRulesClient := ascacherules.New(contextMakerClient)

	// cache data in MongoDB
	cacheClient := ascache.New(cacheRulesClient, service.collection, ascache.WithIgnoreHeaders())

	// Traverse hash values within documents
	hashClient := ashash.New(cacheClient)

	// Save references to the final (hash) client and the cache client to the service.
	service.innerClient = hashClient
	service.cacheClient = cacheClient

	// This is breaking somehow.  Test thoroughly before re-enabling.
	// writableCache := ascache.New(contextMakerClient, collection, ascache.WithWriteOnly())
	// crawlerClient := ascrawler.New(writableCache, ascrawler.WithMaxDepth(4))
	// readOnlyCache := ascache.New(crawlerClient, collection, ascache.WithReadOnly())
	// factory.activityService.Refresh(readOnlyCache, mongodb.NewCollection(collection))
}

func (service *ActivityStream) getClient() streams.Client {

	if service.innerClient == nil {
		service.initClients()
	}

	return service.innerClient
}

func (service *ActivityStream) CacheClient() *ascache.Client {

	if service.cacheClient == nil {
		service.initClients()
	}

	return service.cacheClient
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

// Load implements the Hannibal `Client` interface, and returns a streams.Document from the cache.
func (service *ActivityStream) Load(url string, options ...any) (streams.Document, error) {

	const location = "service.ActivityStream.Load"

	if url == "" {
		return streams.NilDocument(), derp.NotFoundError(location, "Empty URL", url)
	}

	// NPE Check
	client := service.getClient()

	// Forward request to inner client
	result, err := client.Load(url, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Error loading document from inner client", url)
	}

	result.WithOptions(streams.WithClient(service))
	return result, nil
}

// Put adds a single document to the ActivityStream cache
func (service *ActivityStream) Put(document streams.Document) {
	service.CacheClient().Put(document)
}

// Delete removes a single document from the database by its URL
func (service *ActivityStream) Delete(url string) error {

	const location = "service.ActivityStream.Delete"

	if err := service.CacheClient().Delete(url); err != nil {
		return derp.Wrap(err, location, "Error deleting document from cache", url)
	}

	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// PurgeCache removes all expired documents from the cache
func (service *ActivityStream) PurgeCache() error {

	// NPE Check
	if service.collection == nil {
		return derp.InternalError("service.ActivityStream.PurgeCache", "Document Collection not initialized")
	}

	// Purge all expired Documents
	criteria := exp.LessThan("expires", time.Now().Unix())
	collection := mongodb.NewCollection(service.collection)
	if err := collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.ActivityStream.PurgeCache", "Error purging documents")
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStream) queryByRelation(relationType string, relationHref string, cutType string, cutDate int64, done <-chan struct{}) <-chan streams.Document {

	const location = "service.ActivityStream.QueryRelated"

	result := make(chan streams.Document)

	go func() {

		defer close(result)

		// NPE Check
		if service.collection == nil {
			derp.Report(derp.InternalError(location, "Document Collection not initialized"))
			return
		}

		// Build the query
		criteria := exp.
			Equal("metadata.relationType", relationType).
			AndEqual("metadata.relationHref", relationHref)

		var sortOption option.Option

		if cutType == "before" {
			criteria = criteria.AndLessThan("published", cutDate)
			sortOption = option.SortDesc("published")
		} else {
			criteria = criteria.AndGreaterThan("published", cutDate)
			sortOption = option.SortAsc("published")
		}

		// Try to query the database
		documents, err := service.documentIterator(criteria, sortOption)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error querying database"))
			return
		}

		defer documents.Close()

		// Write documents into the result channel until done (or done)
		value := ascache.NewValue()
		for documents.Next(&value) {

			select {
			case <-done:
				return

			default:
				result <- streams.NewDocument(
					value.Object,
					streams.WithHTTPHeader(value.HTTPHeader),
					streams.WithStats(value.Statistics),
					streams.WithClient(service),
				)
			}

			value = ascache.NewValue()
		}
	}()

	return result

}

func (service *ActivityStream) NewDocument(document map[string]any) streams.Document {
	return streams.NewDocument(document, streams.WithClient(service))
}

func (service *ActivityStream) SearchActors(queryString string) ([]model.ActorSummary, error) {

	const location = "service.ActivityStream.SearchActors"

	// If we think this is an address we can work with (because sherlock says so)
	// the try to retrieve it directly.
	if sherlock.IsValidAddress(queryString) {

		// Try to load the actor directly from the Interwebs
		if newActor, err := service.Load(queryString, sherlock.AsActor()); err == nil {

			// If this is a valid, but (previously) unknown actor, then add it to the results
			// This will also automatically get cached/crawled for next time.
			result := []model.ActorSummary{{
				ID:       newActor.ID(),
				Type:     newActor.Type(),
				Name:     newActor.Name(),
				Icon:     newActor.Icon().Href(),
				Username: newActor.PreferredUsername(),
			}}

			return result, nil
		}
	}

	// Fall through means that we can't find a perfect match, so fall back to a full-text search
	collection := mongodb.NewCollection(service.collection)
	result, err := queries.SearchActivityStreamActors(context.TODO(), collection, queryString)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	return result, nil
}

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStream) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation("Reply", inReplyTo, "before", maxDate, done)
}

// QueryRepliesAfterDate returns a slice of streams.Document values that are replies to the specified document, and were published after the specified date.
func (service *ActivityStream) QueryRepliesAfterDate(inReplyTo string, minDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation("Reply", inReplyTo, "after", minDate, done)
}

func (service *ActivityStream) QueryAnnouncesBeforeDate(relationHref string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(vocab.ActivityTypeAnnounce, relationHref, "before", maxDate, done)
}

func (service *ActivityStream) QueryLikesBeforeDate(relationHref string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(vocab.ActivityTypeLike, relationHref, "before", maxDate, done)
}

// SendMessage sends an ActivityPub message to a single recipient/inboxURL
// `inboxURL` the URL to deliver the message to
// `actorType` the type of actor that is sending the message (User, Stream, Search)
// `actorID` unique ID of the actor (zero value for Search Actor)
// `message` the ActivityPub message to send
func (service *ActivityStream) SendMessage(args mapof.Any) error {

	const location = "service.ActivityStream.SendMessage"

	// Collect arguments
	recipientID := args.GetString("to")

	if recipientID == "" {
		return derp.NotFoundError(location, "Recipient ID is required", recipientID)
	}

	message := args.GetMap("message")

	if message.IsEmpty() {
		return derp.NotFoundError(location, "Message is required", message)
	}

	// Find ActivityPub Actor
	actor, err := service.locatorService.GetActor(args.GetString("actorType"), args.GetString("actorID"))

	if err != nil {
		return derp.Wrap(err, location, "Error finding ActivityPub Actor")
	}

	// Send the message to the recipientID
	if err := actor.SendOne(recipientID, message); err != nil {
		return derp.Wrap(err, location, "Error sending message", message, derp.WithCode(http.StatusInternalServerError))
	}

	// Success!!
	return nil
}

func (service *ActivityStream) GetRecipient(recipient string) (string, string, error) {

	const location = "service.ActivityStream.GetRecipient"

	// Try to load the recipient as a JSON-LD document
	document, err := service.Load(recipient, sherlock.AsActor())

	if err != nil {
		return "", "", derp.Wrap(err, location, "Error loading ActivityPub Actor", recipient)
	}

	if !document.IsActor() {
		return "", "", derp.NotFoundError(location, "Recipient is not an ActivityPub Actor", recipient)
	}

	// Successssssssss.
	return document.ID(), document.Inbox().String(), nil
}

func (service *ActivityStream) PublicKeyFinder(keyID string) (string, error) {

	const location = "service.ActivityStream.PublicKeyFinder"

	actorID, _, _ := strings.Cut(keyID, "#")

	actor := service.NewDocument(mapof.Any{
		vocab.PropertyID: actorID,
	})

	// Load the Actor from the document
	actor, err := actor.Load(sherlock.AsActor())

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving Actor from ActivityPub document", actor.Value())
	}

	// Search the Actor's public keys for the one that matches the provided keyID
	for key := range actor.PublicKey().Range() {

		if key.ID() == keyID {
			return key.PublicKeyPEM(), nil
		}
	}

	return "", derp.NotFoundError(location, "Public Key not found", keyID)
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (service *ActivityStream) documentIterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {

	const location = "service.ActivityStream.documentIterator"

	// NPE Check
	if service.collection == nil {
		return nil, derp.InternalError(location, "Document Collection not initialized")
	}

	// Forward request to collection
	collection := mongodb.NewCollection(service.collection)
	return collection.Iterator(criteria, options...)
}
