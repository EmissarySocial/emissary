package ashash

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
)

// Client is a streams.Client wrapper that searches for hash values in a document.
type Client struct {
	innerClient streams.Client
}

// New creates a fully initialized Client object
func New(innerClient streams.Client) *Client {

	result := &Client{
		innerClient: innerClient,
	}

	result.innerClient.SetRootClient(result)
	return result
}

func (client Client) SetRootClient(rootClient streams.Client) {
	client.innerClient.SetRootClient(rootClient)
}

// Load retrieves a document from the underlying innerClient, then searches for hash values
// inside it (if required)
func (client Client) Load(url string, options ...any) (streams.Document, error) {

	// Try to find a hash in the URL
	baseURL, hash, found := strings.Cut(url, "#")

	// If there is no hash, then proceed as is.
	if !found {
		return client.innerClient.Load(url, options...)
	}

	// Otherwise, try to load the baseURL and find the hash inside that document
	result, err := client.innerClient.Load(baseURL, options)

	if err != nil {
		return result, err
	}

	// Search all properties at the top level of the document (not recursive)
	// and scan through arrays (if present) looking for an ID that matches the original URL (base + hash)
	for _, key := range result.MapKeys() {
		for property := result.Get(key); property.NotNil(); property = property.Tail() {
			if property.ID() == url {
				return property, nil
			}
		}
	}

	// Not found.
	return streams.NilDocument(), derp.NotFoundError("ashash.Client.Load", "Hash value not found in document", baseURL, hash, result.Value())
}

func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}
