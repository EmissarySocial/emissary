package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// CrawlDownReplyTree crawls all replies of an ActivityStream document
func CrawlDownReplyTree(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.CrawlDownReplyTree"

	// Collect arguments
	url := args.GetString("url")
	depth := args.GetInt("depth")

	// RULE: The depth cap makes runaway recursion impossible, cycles or not.
	// This runs before any I/O so a capped task costs nothing.
	if depth >= maxCrawlDepth {
		return queue.Success()
	}

	// Get an ActivityStreams client for the whole application
	client := factory.ActivityStream().AppClient()

	// Try to load the document (probably from the cache)
	document, err := client.Load(url)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Loading document"))
	}

	// The seed task from CrawlUpReplyTree carries "force" because the up-crawl's own
	// Load has JUST cached this URL -- without the exemption every crawl would die here
	// at its seed.
	if !args.GetBool("force") {

		// RULE: If we have already seen this document, then
		// we can stop crawling down this branch.
		if ascache.FromCache(document) {
			return queue.Success()
		}
	}

	// Try to load the "replies" collection (probably NOT in the cache)
	replies, err := document.Replies().Load()

	if err != nil {
		return requeue(derp.Wrap(err, location, "Loading replies"))
	}

	// Exit if the replies object isn't actually a collection
	if replies.NotCollection() {
		return queue.Success()
	}

	// Enqueue tasks to index each reply (and their replies).  The signature collapses
	// concurrent duplicates: only one queued task per URL at a time, so overlapping
	// crawls of the same thread cannot multiply.
	for reply := range collections.RangeDocuments(replies) {

		factory.Queue().NewTask(
			"CrawlDownReplyTree",
			mapof.Any{
				"hostname": factory.Hostname(),
				"url":      reply.ID(),
				"depth":    depth + 1,
			},
			queue.WithSignature("CrawlDownReplyTree:"+factory.Hostname()+":"+reply.ID()),
		)
	}

	// No error => success!
	return queue.Success()
}
