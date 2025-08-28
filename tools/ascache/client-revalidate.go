package ascache

import (
	"github.com/benpate/derp"
)

// Revalidate reloads a document from the source even if it has not yet expired.
// This potentially updates the cache timeout value, keeping the document
// "fresh" in the cache for longer.
func (client *Client) Revalidate(url string, options ...any) error {

	const location = "tools.ascache.client.Revalidate"

	// Retrieve the requested document from the inner client
	result, err := client.innerClient.Load(url, options...)

	if err != nil {
		return derp.Wrap(err, location, "Error loading document from inner client", url)
	}

	// Connect to the database
	ctx, cancel := timeoutContext(60)
	defer cancel()

	// Save the updated document to the database
	value := asValue(result)
	if err := client.save(ctx, url, &value); err != nil {
		return derp.Wrap(err, location, "Unable to save revalidated document", url)
	}

	return nil
}
