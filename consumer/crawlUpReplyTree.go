package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// CrawlUpReplyTree crawls ActivityStream documents by traversing the "InReplyTo" property
// until it reaches the top or finds a document that we have already seen. Then, it triggers
// a CrawlDownReplyTree to backfill replies from the top down.
func CrawlUpReplyTree(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.CrawlUpReplyTree"

	// Collect arguments
	url := args.GetString("url")

	// Get an ActivityStreams client for the whole application
	client := factory.ActivityStream().AppClient()

	// Try to load the document (probably from the cache)
	document, err := client.Load(url)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Unable to load document"))
	}

	// If this document is NOT already in the cache, then keep crawling UP the tree
	if !ascache.FromCache(document) {

		// If the loaded document also has an InReplyTo property then continue crawling UP the tree
		if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {

			// Then queue up another task to crawl higher up the tree
			factory.Queue().NewTask(
				"CrawlUpReplyTree",
				mapof.Any{"url": inReplyTo},
			)

			return queue.Success()
		}
	}

	// Otherwise, we have reached the top of the reply tree,
	// so try to crawl DOWN through its replies
	factory.Queue().NewTask(
		"CrawlDownReplyTree",
		mapof.Any{"url": url},
	)

	// Success!
	return queue.Success()
}
