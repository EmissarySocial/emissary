package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// CrawlDownReplyTree crawls all replies of an ActivityStream document
func CrawlDownReplyTree(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.CrawlDownReplyTree"

	url := args.GetString("url")

	client := factory.ActivityStream().AppClient()

	// Try to load the document (probably from the cache)
	document, err := client.Load(url)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Unable to load document"))
	}

	/*
		// If we have already seen this document, then
		// we can stop crawling down this branch.
		if ascache.FromCache(document) {
			return queue.Success()
		}
	*/

	// Try to load the "replies" collection (probably NOT in the cache)
	replies, err := document.Replies().Load()

	if err != nil {
		return requeue(derp.Wrap(err, location, "Unable to load replies"))
	}

	// Exit if the replies object isn't actually a collection
	if replies.NotCollection() {
		return queue.Success()
	}

	// Enqueue tasks to index each reply (and their replies)
	for reply := range collections.RangeDocuments(replies) {

		factory.Queue().NewTask(
			"CrawlDownReplyTree",
			mapof.Any{"url": reply.ID()},
		)
	}

	// No error => success!
	return queue.Success()
}
