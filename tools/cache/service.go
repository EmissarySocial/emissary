package cache

import (
	"encoding/json"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
)

type Client struct {
	collection     data.Collection
	innerClient    streams.Client
	timeoutSeconds int64
	purgeFrequency int64
}

// NewClient returns a fully initialized Client object
func NewClient(collection data.Collection, innerClient streams.Client, options ...OptionFunc) *Client {

	// Create a default client
	result := Client{
		collection:     collection,
		innerClient:    innerClient,
		timeoutSeconds: 60 * 60 * 24, // Default timeout is 24 hours
		purgeFrequency: 60 * 60 * 4,  // Default purge frequency is 4 hours
	}

	// Apply option functions to the client
	for _, option := range options {
		option(&result)
	}

	go result.start()

	return &result
}

// start is a background process that purges expired documents from the cache
func (client *Client) start() {
	for client.purgeFrequency > 0 {
		time.Sleep(time.Second * time.Duration(client.purgeFrequency))

		if err := client.collection.HardDelete(exp.LessThan("exp", time.Now().Unix())); err != nil {
			derp.Report(derp.Wrap(err, "cache.Client.delete", "Error purging expired documents from cache"))
		}
	}
}

func (client *Client) Load(uri string) (streams.Document, error) {

	// Search the cache for the document
	cachedValue := NewCachedValue()
	if err := client.loadByURI(uri, &cachedValue); err == nil {
		value := mapof.NewAny()
		if err := json.Unmarshal([]byte(cachedValue.JSONLD), &value); err != nil {
			return streams.NewDocument(value, streams.WithClient(client)), nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.Load(uri)

	if err != nil {
		return result, derp.Wrap(err, "cache.Client.Load", "error loading document from inner client", uri)
	}

	// Try to save the new value asynchronously
	go client.save(result)

	return result, nil
}

func (client *Client) PurgeByURI(uri string) error {

	if err := client.collection.HardDelete(exp.Equal("uri", uri)); err != nil {
		return derp.Wrap(err, "cache.Client.delete", "Error deleting document from cache (by URI)", uri)
	}

	return nil
}

func (client *Client) save(document streams.Document) {

	// Marshal the document value into JSON.  No, we don't want to save it as BSON.
	marshalledJSON, err := json.Marshal(document.Value())

	if err != nil {
		derp.Report(derp.Wrap(err, "cache.Client.save", "Error marshalling document to JSON", document.Value()))
		return
	}

	// Create a new cachedValue
	cachedValue := NewCachedValue()
	cachedValue.URI = document.ID()
	cachedValue.JSONLD = string(marshalledJSON)
	cachedValue.ExpirationDate = time.Now().Add(time.Second * time.Duration(client.timeoutSeconds)).Unix()

	// Save it to the cache
	if err := client.collection.Save(&cachedValue, "Added to cache"); err != nil {
		derp.Report(derp.Wrap(err, "cache.Client.save", "Error saving document to cache", document.ID(), marshalledJSON))
	}
}

/******************************************
 * Queries
 ******************************************/

func (client Client) loadByURI(uri string, document *CachedValue) error {
	now := time.Now().Unix()
	criteria := exp.Equal("uri", uri).AndGreaterThan("exp", now)
	return client.collection.Load(criteria, document)
}
