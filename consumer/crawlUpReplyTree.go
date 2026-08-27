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
		return requeue(derp.Wrap(err, location, "Loading document"))
	}

	// If this document is NOT already in the cache, then keep crawling UP the tree
	if !ascache.FromCache(document) {

		// RULE: ...but only up to the depth cap.  A hostile or broken server can serve an
		// endless chain of fresh "inReplyTo" URLs; past the cap we settle for crawling
		// down from wherever the climb has reached.
		if depth := args.GetInt("depth"); depth < maxCrawlDepth {

			// If the loaded document also has an InReplyTo property then continue crawling UP the tree
			if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {

				// Then queue up another task to crawl higher up the tree.  The signature
				// collapses concurrent climbs through the same document.
				factory.Queue().NewTask(
					"CrawlUpReplyTree",
					mapof.Any{
						"hostname": factory.Hostname(),
						"url":      inReplyTo,
						"depth":    depth + 1,
					},
					queue.WithSignature("CrawlUpReplyTree:"+factory.Hostname()+":"+inReplyTo),
				)

				return queue.Success()
			}
		}
	}

	// Otherwise, we have reached the top of the reply tree (or the depth cap), so try to
	// crawl DOWN through its replies.  "force" exempts this seed from the down-crawl's
	// already-cached guard: the Load above has just cached this URL, and without the
	// exemption every crawl would stop dead at its seed.
	factory.Queue().NewTask(
		"CrawlDownReplyTree",
		mapof.Any{
			"hostname": factory.Hostname(),
			"url":      url,
			"force":    true,
		},
		queue.WithSignature("CrawlDownReplyTree:"+factory.Hostname()+":"+url),
	)

	// Success!
	return queue.Success()
}
