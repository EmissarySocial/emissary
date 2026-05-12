package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
)

// CrawlContext attempts to backfill the cache for a given document by crawling all of the links
// in its "context" property. If there is an error, this consumer will fall back to crawling the
// "InReplyTo" chain, if that exists.
func CrawlContext(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.CrawlContext"

	// Collect parameters
	objectID := args.GetString("url")

	// Get an ActivityStreams client for the whole application
	client := factory.ActivityStream().AppClient()

	// Try to load the document (probably from the cache)
	document, err := client.Load(objectID)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Unable to load document"))
	}

	// Start first with the document's Context property
	if contextID := document.Context(); contextID != "" {

		// Guarantee that we have a valid URL
		if uri.IsValidURL(contextID) {

			// Load the context collection (probably from the Interweb)
			context, err := client.Load(contextID)

			// If the context is a valid collection, then continue!
			if context.IsCollection() {
				return backfillContext_Context(factory, context)
			}

			// If "too many requests" then requeue later
			if isTooMany, delay := derp.IsTooManyRequests(err); isTooMany {
				return queue.Requeue(delay)
			}

			// Other errors are ignored
			derp.Report(derp.Wrap(err, location, "Unable to load context collection"))
		}
	}

	// Otherwise, try to crawl the InReplyTo tree
	if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {

		factory.Queue().NewTask(
			"CrawlUpReplyTree",
			mapof.Any{"url": inReplyTo},
		)
	}

	// No error => success!
	return queue.Success()
}

// backfillContext_Context indexes every document in the collection that we haven't already seen
func backfillContext_Context(factory *service.Factory, context streams.Document) queue.Result {

	// Scan all documents in the collection until we find one we've already seen...
	for document := range collections.RangeDocuments(context) {

		// Try to load the complete document from the Interweb
		document, err := context.Load(document.ID())

		// If there was an error loading a specific document...
		if err != nil {

			// Retry (using queue this time) in one hour
			factory.Queue().NewTask(
				"ReindexActivityStream",
				mapof.Any{
					"host": factory.Hostname(),
					"url":  document.ID(),
				},
				queue.WithDelayHours(1),
			)

			// ... but don't stop processing the rest of the documents in the context
			continue
		}

		// If we have already cached this document, then we are up to date; exit
		if ascache.FromCache(document) {
			break
		}
	}

	// Woot.
	return queue.Success()
}
