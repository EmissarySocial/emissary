package ascache

import (
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
)

type Client struct {
	collection     data.Collection
	innerClient    streams.Client
	expireSeconds  int64
	purgeFrequency int64
}

// New returns a fully initialized Client object
func New(collection data.Collection, innerClient streams.Client, options ...OptionFunc) *Client {

	// Create a default client
	result := Client{
		collection:     collection,
		innerClient:    innerClient,
		expireSeconds:  60 * 60 * 24 * 30, // Default expiration is 30 days
		purgeFrequency: 60 * 60 * 4,       // Default purge frequency is 4 hours
	}

	// Apply option functions to the client
	for _, option := range options {
		option(&result)
	}

	go result.start()

	return &result
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// start is a background process that purges expired documents from the cache
func (client *Client) start() {

	if client.collection == nil {
		return
	}

	for client.purgeFrequency > 0 {
		time.Sleep(time.Second * time.Duration(client.purgeFrequency))

		if err := client.collection.HardDelete(exp.LessThan("exp", time.Now().Unix())); err != nil {
			derp.Report(derp.Wrap(err, "cache.Client.delete", "Error purging expired documents from cache"))
		}
	}
}

/******************************************
 * Hannibal HTTP Client Methods
 ******************************************/

func (client *Client) Load(uri string) (streams.Document, error) {

	// Search the cache for the document
	if client.collection != nil {
		cachedValue := NewCachedValue()
		if err := client.loadByURI(uri, &cachedValue); err == nil {

			if cachedValue.ShouldRefresh() {
				go client.refresh(uri, cachedValue)
			}

			return streams.NewDocument(cachedValue.Original, streams.WithClient(client)), nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.Load(uri)

	if err != nil {
		return result, derp.Wrap(err, "cache.Client.Load", "error loading document from inner client", uri)
	}

	// Try to save the new value asynchronously
	if client.collection != nil {
		go client.save(uri, result)
	}

	result.WithOptions(streams.WithClient(client))

	return result, nil
}

/******************************************
 * Other Cache Management Methods
 ******************************************/

func (client *Client) PurgeByURI(uri string) error {

	if client.collection == nil {
		return derp.NewInternalError("cache.Client.delete", "Cache connection is not defined")
	}

	if err := client.collection.HardDelete(exp.Equal("uri", uri)); err != nil {
		return derp.Wrap(err, "cache.Client.delete", "Error deleting document from cache (by URI)", uri)
	}

	return nil
}

func (client *Client) refresh(uri string, value CachedValue) {

	if client.collection == nil {
		return
	}

	// Pass the request to the inner client
	if result, err := client.innerClient.Load(uri); err == nil {
		client.save(uri, result)
	}
}

func (client *Client) save(uri string, document streams.Document) {

	if client.collection == nil {
		return
	}

	// Create a new cachedValue
	cachedValue := NewCachedValue()
	cachedValue.URI = uri
	cachedValue.Original = document.Map()
	cachedValue.PublishedDate = document.Published().Unix()

	// TODO: LOW: should see if the document has a header that defines the cache duration
	cachedValue.ExpiresDate = time.Now().Add(time.Second * time.Duration(client.expireSeconds)).Unix()
	cachedValue.RefreshesDate = CalcRefreshDate(time.Now().Unix(), cachedValue.ExpiresDate)

	if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
		cachedValue.InReplyTo = inReplyTo.String()
	}

	// Save it to the cache
	if err := client.collection.Save(&cachedValue, ""); err != nil {
		derp.Report(derp.Wrap(err, "cache.Client.save", "Error saving document to cache", document.ID()))
	}

	// If this is a reply, then cache the parent document as well
	if cachedValue.InReplyTo != "" {
		go client.Load(cachedValue.InReplyTo)
	}
}

/******************************************
 * Other Queries
 ******************************************/
func (client *Client) loadByURI(uri string, document *CachedValue) error {

	if client.collection == nil {
		return derp.NewInternalError("cache.Client.loadByURI", "Cache connection is not defined")
	}

	criteria := exp.Equal("uri", uri)
	err := client.collection.Load(criteria, document)

	return err
}
