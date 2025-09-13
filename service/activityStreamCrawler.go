package service

import (
	"slices"

	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ascrawler"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityStreamCrawler struct {
	client    streams.Client
	enqueue   chan<- queue.Task
	hostname  string
	actorType string
	actorID   primitive.ObjectID
	maxDepth  int
}

func NewActivityStreamCrawler(client streams.Client, enqueue chan<- queue.Task, hostname string, actorType string, actorID primitive.ObjectID, maxDepth int) ActivityStreamCrawler {
	return ActivityStreamCrawler{
		client:    client,
		enqueue:   enqueue,
		hostname:  hostname,
		actorType: actorType,
		actorID:   actorID,
		maxDepth:  maxDepth,
	}
}

// Crawl is a public-facing function that crawls a given URL and its linked documents.
// It can potentially take a lot of time to complete, so it should only be called via
// a queued task.
func (service *ActivityStreamCrawler) Crawl(url string, history []string) error {

	const location = "service.ActivityStreamCrawler.Crawl"

	// Do not crawl beyond the maximum depth
	if len(history) > service.maxDepth {
		return nil
	}

	// Load the document from the Interwebs
	document, err := service.client.Load(url, ascrawler.WithoutCrawler())

	if err != nil {
		return derp.Wrap(err, location, "Unable to load ActivityStreams document")
	}

	// Crawl the document's contents
	service.crawl(document, history)
	return nil
}

// crawl is the inner crawler function that does the majority of the work. It requires
// a fully populates ActivityStreams document.  It can potentially take a lot of time
// to complete, so it should only be called via a queued task or a goroutine.
func (service *ActivityStreamCrawler) crawl(document streams.Document, history []string) {

	const location = "service.ActivityService.crawl"

	// Collect the URL that we're going to crawl
	documentID := document.ID()

	if documentID == "" {
		log.Trace().Msg("Crawler skipping because document has no ID")
		return
	}

	// Do not crawl beyond the maximum depth
	historyLength := len(history)

	if historyLength > service.maxDepth {
		log.Trace().Int("depth", historyLength).Int("maxDepth", service.maxDepth).Msg("Crawler skipping because document has exceeded max depth")
		return
	}

	// Find the depth of the currently cached document
	if depthString := document.HTTPHeader().Get(headerCrawlerDepth); depthString != "" {

		currentDepth := convert.Int(depthString)

		// If the cached document's depth is LESS OR EQUAL to our history length, then don't re-crawl it.
		if currentDepth <= historyLength {
			log.Trace().Int("depth", historyLength).Int("maxDepth", service.maxDepth).Msg("Crawler skipping because document has already been found (at a lower depth)")
			return
		}

		// Otherwise, update the cache with the new (lower) depth
		document.HTTPHeader().Set(headerCrawlerDepth, convert.String(historyLength))

		if err := service.client.Save(document); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to update document with lower historyLength", documentID))
			// Continue processing even if this fails.
		}
	}

	// Add the document ID to the history for all future crawls
	history = append(history, documentID)

	log.Debug().Str("loc", location).Str("url", documentID).Int("depth", len(history)).Msg("Crawling document")

	// Crawl `AttributedTo` property
	for attributedTo := range document.AttributedTo().Range() {
		if attributedTo := attributedTo.ID(); attributedTo != "" {
			service.sendTask(attributedTo, history)
		}
	}

	// Crawl `InReplyTo` property
	if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {
		service.sendTask(inReplyTo, history)
	}

	// Crawl `Context` property
	service.crawl_Collection(document, vocab.PropertyContext, history)

	// Crawl `Replies` property
	service.crawl_Collection(document, vocab.PropertyReplies, history)

	// Crawl `Likes` property
	service.crawl_Collection(document, vocab.PropertyLikes, history)

	// Crawl `Shares` property
	service.crawl_Collection(document, vocab.PropertyShares, history)
}

// crawl_Collection loads/caches all documents in a collection, continuing until a
// previously-cached value is found.  Because of this, it must load each document
// sequentially, and therefore may take a long time to complete.
func (service *ActivityStreamCrawler) crawl_Collection(document streams.Document, propertyName string, history []string) {

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

	// If the result document is not a collection, then we cannot crawl it
	if !collection.IsCollection() {
		return
	}

	// Crawl first 2048 documents in the collection
	documents := collections.RangeDocuments(collection) // Recurse through all pages in the collection
	documents2048 := ranges.Limit(2048, documents)      // Scan 2048 collection items at most

	for item := range documents2048 {

		// This is bad.  Why would you do this?
		if item.ID() == "" {
			continue
		}

		// Load the document from the rootClient (which should also cache it recursively)
		document, err := service.client.Load(item.ID(), ascrawler.WithoutCrawler())

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to load ActivityStream"))
		}

		// If we've found an item that's already in the cache, then we've reached
		// the end of the "new" items in this collection. So we can stop here.
		if document.HTTPHeader().Get(ascache.HeaderHannibalCache) == "true" {
			break
		}

		service.sendTask(document.ID(), history)
	}
}

func (service *ActivityStreamCrawler) sendTask(url string, history []string) {

	const location = "service.ActivityService.sendTask"

	// RULE: URL must be a valid URL
	if !isValidURL(url) {
		return
	}

	// RULE: Current crawler depth cannot exceed maximum
	if len(history) >= service.maxDepth {
		return
	}

	// RULE: URL must not be in direct history (to prevent cycles)
	if slices.Contains(history, url) {
		return
	}

	// Calculate delay based on history length (in seconds)
	delay, priority := service.calcDelayAndPriority(len(history))

	service.enqueue <- queue.NewTask(
		"CrawlActivityStreams",
		mapof.Any{
			"host":      service.hostname,
			"actorType": service.actorType,
			"actorID":   service.actorID,
			"url":       url,
			"history":   history,
		},
		queue.WithPriority(priority),  // medium priority background process
		queue.WithDelaySeconds(delay), // wait one minute (to catch duplicates and prevent spam)
		queue.WithSignature(url),      // URL helps prevent duplicate calls
	)

	// Done!
	log.Debug().Str("loc", location).Str("url", url).Int("depth", len(history)).Msg("Task queued")
}

func (service *ActivityStreamCrawler) calcDelayAndPriority(historyLength int) (delaySeconds int, priority int) {

	switch historyLength {

	case 0:
		return 0, 32 // Run immediately but write to the database first

	case 1:
		return 0, 64 // Run immediately but write to the database first

	case 2:
		return 20, 128 // Wait 20 seconds, low priority

	case 3:
		return 40, 256 // Wait 40 seconds, low priority

	default:
		return 60, 512 // Wait 1 minute, low priority
	}
}
