package ascrawler

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

type Client struct {
	queue       *queue.Queue
	innerClient streams.Client
	maxDepth    int
	hostname    string
}

func New(queue *queue.Queue, innerClient streams.Client, hostname string, options ...ClientOption) *Client {

	// Create client
	result := &Client{
		queue:       queue,
		innerClient: innerClient,
		maxDepth:    4,
		hostname:    hostname,
	}

	// Apply options
	for _, option := range options {
		option(result)
	}

	// Pass the root client down into the innerClient
	result.innerClient.SetRootClient(result)

	return result
}

func (client *Client) SetRootClient(rootClient streams.Client) {
	client.innerClient.SetRootClient(rootClient)
}

func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	const location = "tools.ascrawler.Load"

	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, derp.Wrap(err, location, "Error loading actor from inner client")
	}

	go client.crawl(result.Clone(), options...)

	return result, nil
}

// crawl is the main recursive loop. It looks for crawl-able properties in the document
// and loads them into the cache.
func (client *Client) crawl(document streams.Document, options ...any) {

	const location = "tools.ascrawler.crawl"

	config := parseLoadConfig(options...)

	// Prevent infinite loops....
	if config.currentDepth >= client.maxDepth {
		return
	}

	log.Debug().Str("loc", location).Str("url", document.ID()).Msg("Loading related documents")

	client.crawl_AttributedTo(document, config)
	client.crawl_Context(document, config)
	client.crawl_InReplyTo(document, config)
	client.crawl_Replies(document, config)
}

func (client Client) crawl_AttributedTo(document streams.Document, config loadConfig) {

	for attributedTo := range document.AttributedTo().Range() {
		if url := attributedTo.ID(); url != "" {
			client.sendTask(url, config.currentDepth+1)
		}
	}
}

func (client Client) crawl_Context(document streams.Document, config loadConfig) {
	client.crawl_Collection(document, vocab.PropertyContext, config)
}

func (client Client) crawl_InReplyTo(document streams.Document, config loadConfig) {
	if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {
		client.sendTask(inReplyTo, config.currentDepth)
	}
}

func (client Client) crawl_Replies(document streams.Document, config loadConfig) {
	client.crawl_Collection(document, vocab.PropertyReplies, config)
}

func (client Client) crawl_Collection(document streams.Document, propertyName string, config loadConfig) {

	// Get the designated property from the document
	collection, err := document.Get(propertyName).Load()

	if err != nil {
		derp.Report(derp.Wrap(err, "ascrawler.crawlCollection", "Error loading collection", propertyName))
		return
	}

	// Crawl first 2048 documents in the collection
	done := make(chan struct{})
	documents := collections.Documents(collection, done)
	documents2048 := channel.Limit(2048, documents, done)

	for document := range documents2048 {
		if url := document.ID(); url != "" {
			client.sendTask(url, config.currentDepth+1)
		}
	}
}

func (client Client) sendTask(url string, depth int) {

	const location = "tools.ascrawler.sendTask"

	task := queue.NewTask(
		"CrawlActivityStreams",
		mapof.Any{
			"host":  client.hostname,
			"url":   url,
			"depth": depth,
		},
	)

	if err := client.queue.Publish(task); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to publish task to queue"))
		return
	}

	log.Debug().Str("loc", location).Str("url", url).Int("depth", depth).Msg("Published task to queue")
}
