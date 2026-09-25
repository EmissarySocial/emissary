package consumer

import (
	"time"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
)

// CrawlContext backfills the cache for a document by crawling the links in its "context"
// property, falling back to the "InReplyTo" chain when the context cannot be read
func CrawlContext(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.CrawlContext"

	// Collect parameters
	objectID := args.GetString("url")

	// Get an ActivityStreams client for the whole application
	client := factory.ActivityStream().AppClient()

	// Try to load the document (probably from the cache)
	document, err := client.Load(objectID)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Loading document"))
	}

	// Start first with the document's Context property
	if contextID := document.Context(); contextID != "" {

		// Guarantee that we have a valid URL
		if uri.IsValidURL(contextID) {

			// Load the context collection (probably from the Interweb)
			context, err := client.Load(contextID)

			switch outcome, retryAfter := classifyContext(context, err); outcome {

			// A real collection is the whole point of this task
			case contextOutcomeBackfill:
				return backfillContext_Context(factory, context)

			// A rate limit is the host's throttle, so wait out the host's own Retry-After
			case contextOutcomeRateLimited:

				log.Debug().Str("location", location).Str("context", contextID).
					Dur("retryAfter", retryAfter).Msg("Rate limited by remote host")

				return queue.Requeue(retryAfter)
			}

			// RULE: Every other outcome is the remote's own and repeats identically on every
			// future crawl, so it is logged and never reported (BUG-150).  The InReplyTo crawl
			// below is this task's real answer to a context it cannot read.
			log.Debug().Str("location", location).Str("context", contextID).
				Int("code", derp.ErrorCode(err)).Msg("Skipping unreadable context")
		}
	}

	// Otherwise, try to crawl the InReplyTo tree.  The signature collapses concurrent
	// climbs through the same document (the same thread arriving via many followers).
	if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {

		factory.Queue().NewTask(
			"CrawlUpReplyTree",
			mapof.Any{"hostname": factory.Hostname(), "url": inReplyTo},
			queue.WithSignature("CrawlUpReplyTree:"+factory.Hostname()+":"+inReplyTo),
		)
	}

	// No error => success!
	return queue.Success()
}

// contextOutcome names what a loaded "context" document means for the crawl
type contextOutcome int

const (
	// contextOutcomeSkip means the context is unusable, and the InReplyTo tree is the fallback.
	// It is first so that the zero value does the least, rather than starting a crawl.
	contextOutcomeSkip contextOutcome = iota

	// contextOutcomeBackfill means the context is a real collection, and is worth indexing
	contextOutcomeBackfill

	// contextOutcomeRateLimited means the HOST is throttling us, and nothing here is wrong
	contextOutcomeRateLimited
)

// classifyContext decides what a loaded "context" document means for the crawl, and how long
// to wait before trying again.  The duration is zero for every outcome except RateLimited.
func classifyContext(context streams.Document, err error) (contextOutcome, time.Duration) {

	// RULE: The error MUST be settled before the document is read.  A failed Load still
	// returns a usable Document, so asking IsCollection first answers from an empty value.
	if err != nil {

		// RULE: The 429 test comes first, and its duration is carried out rather than
		// recomputed.  derp reads the host's Retry-After header, falling back to one hour,
		// and never returns zero here -- so this can not spin against a throttling host.
		if isTooMany, retryAfter := derp.IsTooManyRequests(err); isTooMany {
			return contextOutcomeRateLimited, retryAfter
		}

		// RULE: Every other failure is the remote's own and repeats identically on every
		// future crawl.  A context answering HTML is reported as 500, so no class is exempt.
		return contextOutcomeSkip, 0
	}

	// A context that is not a collection is a normal document on the open fediverse,
	// not a failure.  There is simply nothing to backfill from it.
	if !context.IsCollection() {
		return contextOutcomeSkip, 0
	}

	return contextOutcomeBackfill, 0
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
					"hostname": factory.Hostname(),
					"url":      document.ID(),
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
