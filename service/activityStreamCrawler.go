package service

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityStreamCrawler struct {
	client    streams.Client
	queue     *queue.Queue
	hostname  string
	actorType string
	actorID   primitive.ObjectID
}

func NewActivityStreamCrawler(client streams.Client, queue *queue.Queue, hostname string, actorType string, actorID primitive.ObjectID) ActivityStreamCrawler {
	return ActivityStreamCrawler{
		client:    client,
		queue:     queue,
		hostname:  hostname,
		actorType: actorType,
		actorID:   actorID,
	}
}

// Crawl is a public-facing function that crawls a given URL and its linked documents.
// It can potentially take a lot of time to complete, so it should only be called via
// a queued task.
func (service *ActivityStreamCrawler) Crawl(url string) error {

	// Emergency removing crawler.  Will re-implement later.
	return nil

	/*
		const location = "service.ActivityStreamCrawler.Crawl"

		// RULE: URL must not be empty
		if url == "" {
			return nil
		}

		// RULE: URL must be a valid URL
		if !isValidURL(url) {
			return nil
		}

		// Load the document from the Interwebs
		document, err := service.client.Load(url, ascrawler.WithoutCrawler())

		if err != nil {
			return derp.Wrap(err, location, "Unable to load ActivityStreams document")
		}

		// If this is an "Actor" then do not crawl anything else
		// (no "outbox" or "featured" collections)
		if streams.IsActor(document.Type()) {
			return nil
		}

		// If this is an "Activity" then just load the `actor` and `object` properties.
		if streams.IsActivity(document.Type()) {
			service.load(document.Actor().ID())
			service.load(document.Object().ID())
			return nil
		}

		// If this is a collection, then crawl it directly.
		if streams.IsCollection(document.Type()) {
			service.crawl_CollectionDocument(document)
			return nil
		}

		// Otherwise, we're going to crawl a Document (Note, Article, etc.)

		// Load actor(s) from `AttributedTo` property
		for attributedTo := range document.AttributedTo().Range() {
			service.load(attributedTo.ID())
		}

		// Crawl document from `InReplyTo` property
		// This is *probably* also handled by contextMaker, but let's put
		// it here, too, just in case that is removed/changed some day.
		service.load(document.InReplyTo().ID())

		// Crawl `Context` property
		service.crawl_Collection(document, vocab.PropertyContext)

		// Crawl `Replies` property
		service.crawl_Collection(document, vocab.PropertyReplies)

		// Crawl `Likes` property
		service.crawl_Collection(document, vocab.PropertyLikes)

		// Crawl `Shares` property
		service.crawl_Collection(document, vocab.PropertyShares)

		return nil

	*/
}

/*
// crawl_Collection loads/caches all documents in a collection, continuing until a
// previously-cached value is found.  Because of this, it must load each document
// sequentially, and therefore may take a long time to complete.
func (service *ActivityStreamCrawler) crawl_Collection(document streams.Document, propertyName string) {

	const location = "service.ActivityService.crawl_Collection"

	// Get the designated property from the document
	propertyValue := document.Get(propertyName)

	if !isValidURL(propertyValue.ID()) {
		return
	}

	// Get the designated property from the document
	collection, err := service.client.Load(propertyValue.ID())

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load collection", propertyName))
		return
	}

	// Continue to crawl the document itself
	service.crawl_CollectionDocument(collection)
}

// crawl_Collection loads/caches all documents in a collection, continuing until a
// previously-cached value is found.  Because of this, it must load each document
// sequentially, and therefore may take a long time to complete.
func (service *ActivityStreamCrawler) crawl_CollectionDocument(collection streams.Document) {

	// If the document is not a collection, then we cannot crawl it
	if !collection.IsCollection() {
		return
	}

	// Crawl first 2048 documents in the collection
	documents := collections.RangeDocuments(collection) // Recurse through all pages in the collection
	documents2048 := ranges.Limit(2048, documents)      // Scan 2048 collection items at most

	for item := range documents2048 {

		// Load the document from the root client (cache or interwebs)
		document := service.load(item.ID())

		// If we've found an item that's already in the cache, then we've reached
		// the end of the new items in this collection. So we can stop here.
		if document.HTTPHeader().Get(ascache.HeaderHannibalCache) == "true" {
			break
		}
	}
}

func (service *ActivityStreamCrawler) load(url string) streams.Document {

	const location = "service.ActivityService.sendTask"

	// RULE: url must not be empty
	if url == "" {
		return streams.NilDocument()
	}

	// RULE: URL must be a valid URL
	if !isValidURL(url) {
		return streams.NilDocument()
	}

	// Load the URL from the Interwebs, report errors,
	// but do not retry, do not recurse, and do not stop.
	document, err := service.client.Load(url, ascrawler.WithoutCrawler())

	// Report errors, but do not retry
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load URL", url))
		return streams.NilDocument()
	}

	return document
}
*/
