package ascrawler

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/channel"
)

type Client struct {
	innerClient streams.Client
	maxDepth    int
	// TODO: This needs a composable queue runner...
}

func New(innerClient streams.Client, options ...ClientOption) Client {

	result := Client{
		innerClient: innerClient,
		maxDepth:    3,
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

func (client Client) Load(uri string, options ...any) (streams.Document, error) {

	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, derp.Wrap(err, "ascrawler.Load", "Error loading actor from inner client")
	}

	go client.crawl(result, 0) // TODO: this should be a buffered enqueue operation

	return result, nil
}

// crawl is the main recursive loop. It looks for crawl-able properties in the document
// and loads them into the cache.
func (client Client) crawl(document streams.Document, depth int) {

	// Prevent infinite loops....
	if depth >= client.maxDepth {
		return
	}

	// If the document is already cached, then don't crawl it again.
	if cached := document.HTTPHeader().Get("X-Hannibal-Cache"); cached == "true" {
		return
	}

	// Try to load the document then crawl it's linked data
	if loaded, err := document.Load(); err == nil {

		// Crawl Related Documents
		client.crawlDocument(loaded, vocab.PropertyContext, depth)
		client.crawlDocument(loaded, vocab.PropertyInReplyTo, depth)
		client.crawlDocument(loaded, vocab.PropertyAttributedTo, depth)

		// Crawl Related Collections
		client.crawlCollection(loaded, vocab.PropertyReplies, depth)
	}
}

// crawlDocument searches for one or more documents in a single property that can be crawled
func (client Client) crawlDocument(document streams.Document, propertyName string, depth int) {

	// Iterate through (potential) multiple values in the property
	for property := document.Get(propertyName); property.NotNil(); property = property.Tail() {
		go client.crawl(property.Head(), depth+1) // TODO: this should be a buffered enqueue operation
	}
}

// crawlCollection searches for all documents in a collection that can be crawled.
func (client Client) crawlCollection(document streams.Document, propertyName string, depth int) {

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
		go client.crawl(document, depth+1) // TODO: this should be a buffered enqueue operation
	}
}
