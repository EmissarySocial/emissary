package ashash

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/uri"
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

// SetRootClient implements the streams.Client interface, and passes the root client down the chain
func (client Client) SetRootClient(rootClient streams.Client) {
	if client.innerClient != nil {
		client.innerClient.SetRootClient(rootClient)
	}
}

// Load retrieves a document from the underlying innerClient, then searches for hash values
// inside it (if required)
func (client Client) Load(id string, options ...any) (streams.Document, error) {

	const location = "ashash.Client.Load"

	if uri.IsValidURL(id) {

		// If we can find a "hash" in the URL, then run this middleware
		if baseURL, hash, found := strings.Cut(id, "#"); found {

			// Otherwise, try to load the baseURL and find the hash inside that document
			result, err := client.innerClient.Load(baseURL, options...)

			if err != nil {
				return result, derp.Wrap(err, location, "Loading base URL", baseURL)
			}

			// Search all properties at the top level of the document (not recursive)
			// and scan through arrays (if present) looking for an ID that matches the original URL (base + hash)
			for _, key := range result.MapKeys() {
				for property := result.Get(key); property.NotNil(); property = property.Tail() {
					if property.ID() == id {
						return property, nil
					}
				}
			}

			// Inner hashed ID not found.
			return streams.NilDocument(), derp.NotFound(location, "Hash value not found in document", baseURL, hash, result.Value())
		}
	}

	result, err := client.innerClient.Load(id, options...)

	if err != nil {
		return result, derp.Wrap(err, location, "Loading document", id)
	}

	return result, nil
}

// Save implements the streams.Client interface, and passes the document to the innerClient
func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

// Delete implements the streams.Client interface, and passes the documentID to the innerClient
func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}
