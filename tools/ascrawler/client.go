package ascrawler

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Client struct {
	queue       *queue.Queue
	rootClient  streams.Client
	innerClient streams.Client
	maxDepth    int
	actorType   string
	actorID     primitive.ObjectID
	hostname    string
}

func New(queue *queue.Queue, innerClient streams.Client, actorType string, actorID primitive.ObjectID, hostname string, options ...ClientOption) *Client {

	// Create client
	result := &Client{
		queue:       queue,
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

	// Load the actual document from the inner client
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, derp.Wrap(err, location, "Unable to load actor from inner client")
	}

	// Determined if we should crawl related records (default=yes)
	if config := parseLoadConfig(options...); !config.useCrawler {
		return result, nil
	}

	client.queue.NewTask(
		"CrawlActivityStreams",
		mapof.Any{
			"host":      client.hostname,
			"actorType": client.actorType,
			"actorID":   client.actorID,
			"url":       uri,
		},
		queue.WithSignature(uri), // URL prevents duplicate calls
	)

	// Return the result to the caller
	return result, nil
}

func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}
