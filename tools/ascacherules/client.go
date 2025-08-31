package ascacherules

import (
	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
)

type Client struct {
	innerClient streams.Client
}

func New(innerClient streams.Client) *Client {
	result := &Client{
		innerClient: innerClient,
	}

	result.innerClient.SetRootClient(result)
	return result
}

func (client *Client) SetRootClient(rootClient streams.Client) {
	client.innerClient.SetRootClient(rootClient)
}

func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	// Retrieve the actual document from the inner client.
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, err
	}

	header := result.HTTPHeader()
	cacheControl := cacheheader.Parse(header)

	switch {

	// Activity objects are never cached.  This prevents likes,
	// reposts, and other actions from being cached.
	case result.IsActivity():

		cacheControl.MaxAge = 0
		cacheControl.NoStore = true

	// Collections (et al) are cached for up to one minute. This
	// minimizes traffic on heavy loads, but keeps collections
	// refreshed in (near) real-time
	case result.IsCollection():

		cacheControl.MaxAge = clamp(0, cacheControl.MaxAge, minute)

	// Actors are complicated.  IF they include an outbox (a la sherlock/RSS feeds)
	// then we cache them like they ARE a collection.  Otherwise, it's Okay (I think)
	// to cache it for longer
	case result.IsActor():

		// This happens when we're "faking" an actor via RSS
		if result.Outbox().IsMap() {
			cacheControl.MaxAge = clamp(hour, cacheControl.MaxAge, day)

		} else {
			// This is a normal ActivityPub lookup
			cacheControl.MaxAge = clamp(day, cacheControl.MaxAge, month)
		}

	// All other items (Articles, Notes, etc) are cached for up to a year
	default:
		cacheControl.MaxAge = clamp(day, cacheControl.MaxAge, year)
	}

	// Write the cacheControl value back into the document header
	header.Set("Cache-Control", cacheControl.String())
	result.SetHTTPHeader(header)

	// Return the result
	return result, nil
}

func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}
