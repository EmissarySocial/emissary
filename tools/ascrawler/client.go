package ascrawler

import (
	"slices"

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
	rootClient  streams.Client
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
	client.rootClient = rootClient
}

func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	const location = "tools.ascrawler.Load"

	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, derp.Wrap(err, location, "Error loading actor from inner client")
	}

	// Parse the load configuration and determine if we should crawl related records or not
	config := parseLoadConfig(options...)

	if config.useCrawler {

		config.history = append(config.history, uri)

		// Crawl related records if we're below the max depth
		if len(config.history) < client.maxDepth {
			client.crawl_AttributedTo(result, config)
			client.crawl_InReplyTo(result, config)
			client.crawl_Context(result, config)
			client.crawl_Replies(result, config)
			client.crawl_Likes(result, config)
			client.crawl_Shares(result, config)
		}
	}

	return result, nil
}

func (client Client) crawl_AttributedTo(document streams.Document, config loadConfig) {

	for attributedTo := range document.AttributedTo().Range() {
		if url := attributedTo.ID(); url != "" {
			client.sendTask(url, config)
		}
	}
}

func (client Client) crawl_InReplyTo(document streams.Document, config loadConfig) {
	if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {
		client.sendTask(inReplyTo, config)
	}
}

func (client Client) crawl_Context(document streams.Document, config loadConfig) {
	client.crawl_Collection(document, vocab.PropertyContext, config)
}

func (client Client) crawl_Replies(document streams.Document, config loadConfig) {
	client.crawl_Collection(document, vocab.PropertyReplies, config)
}

func (client Client) crawl_Likes(document streams.Document, config loadConfig) {
	client.crawl_Collection(document, vocab.PropertyLikes, config)
}

func (client Client) crawl_Shares(document streams.Document, config loadConfig) {
	client.crawl_Collection(document, vocab.PropertyShares, config)
}

func (client Client) crawl_Collection(document streams.Document, propertyName string, config loadConfig) {

	property := document.Get(propertyName)

	// RULE: If the property is not a valid URL, then there is nothing to load
	if !isValidURL(property.ID()) {
		return
	}

	// Get the designated property from the document
	collection, err := client.rootClient.Load(property.ID())

	if err != nil {
		derp.Report(derp.Wrap(err, "ascrawler.crawlCollection", "Error loading collection", propertyName))
		return
	}

	// If the result document is not a collection, then we cannot crawl it
	if !collection.IsCollection() {
		return
	}

	// Crawl first 2048 documents in the collection
	done := make(chan struct{})
	documents := collections.Documents(collection, done)
	documents2048 := channel.Limit(2048, documents, done)

	for document := range documents2048 {
		if url := document.ID(); url != "" {
			client.sendTask(url, config)
		}
	}
}

func (client Client) sendTask(url string, config loadConfig) {

	const location = "tools.ascrawler.sendTask"

	// RULE: URL must be a valid URL
	if !isValidURL(url) {
		return
	}

	// RULE: Current crawler depth cannot exceed maximum
	if len(config.history) >= client.maxDepth {
		log.Debug().Str("loc", location).Str("url", url).Int("depth", len(config.history)).Msg("Skipping task due to max depth")
		return
	}

	// RULE: URL must not be in direct history (to prevent cycles)
	if slices.Contains(config.history, url) {
		log.Debug().Str("loc", location).Str("url", url).Int("depth", len(config.history)).Msg("Skipping task due to history cycle")
		return
	}

	// Queue the task
	log.Debug().Str("loc", location).Str("url", url).Int("depth", len(config.history)).Msg("Queuing task")

	client.enqueue <- queue.NewTask(
		"CrawlActivityStreams",
		mapof.Any{
			"host":      client.hostname,
			"actorType": client.actorType,
			"actorID":   client.actorID,
			"url":       url,
			"history":   config.history,
		},
		queue.WithPriority(128),   // low priority background process
		queue.WithDelayMinutes(1), // wait one minute (to catch duplicates and prevent spam)
		queue.WithSignature(url),  // URL helps prevent duplicate calls
	)
}
