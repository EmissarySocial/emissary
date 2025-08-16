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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Client struct {
	enqueue     chan<- queue.Task
	innerClient streams.Client
	maxDepth    int
	actorType   string
	actorID     primitive.ObjectID
	hostname    string
}

func New(enqueue chan<- queue.Task, innerClient streams.Client, actorType string, actorID primitive.ObjectID, hostname string, options ...ClientOption) *Client {

	// Create client
	result := &Client{
		enqueue:     enqueue,
		innerClient: innerClient,
		maxDepth:    4,
		actorType:   actorType,
		actorID:     actorID,
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

	config := parseLoadConfig(options...)

	// Crawl parents, children, and related records if we're below the max depth
	if config.currentDepth < client.maxDepth {
		client.crawl_AttributedTo(result, config)
		client.crawl_Context(result, config)
		client.crawl_InReplyTo(result, config)
		client.crawl_Replies(result, config)
	}

	return result, nil
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

	log.Debug().Str("loc", location).Str("url", url).Int("depth", depth).Msg("Queuing task")

	client.enqueue <- queue.NewTask(
		"CrawlActivityStreams",
		mapof.Any{
			"host":      client.hostname,
			"actorType": client.actorType,
			"actorID":   client.actorID,
			"url":       url,
			"depth":     depth,
		},
		queue.WithSignature(url),
		queue.WithPriority(128),
	)
}
