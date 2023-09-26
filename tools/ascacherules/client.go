package ascacherules

import (
	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

type Client struct {
	innerClient streams.Client
}

func New(innerClient streams.Client) Client {
	return Client{
		innerClient: innerClient,
	}
}

func (client Client) Load(uri string, options ...any) (streams.Document, error) {

	// Retrieve the actual document from the inner client.
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, err
	}

	header := result.HTTPHeader()
	cacheControl := cacheheader.Parse(header)

	switch result.Type() {

	// Collections (et al) are cached for up to one minute. This
	// minimizes traffic on heavy loads, but keeps collections
	// refreshed in (near) real-time
	case
		vocab.CoreTypeCollection,
		vocab.CoreTypeCollectionPage,
		vocab.CoreTypeOrderedCollection,
		vocab.CoreTypeOrderedCollectionPage:

		cacheControl.MaxAge = clamp(0, cacheControl.MaxAge, minute)

	// Actors are complicated.  IF they include an outbox (a la sherlock/RSS feeds)
	// then we cache them like they ARE a collection.  Otherwise, it's Okay (I think)
	// to cache it for longer
	case
		vocab.ActorTypeApplication,
		vocab.ActorTypeGroup,
		vocab.ActorTypeOrganization,
		vocab.ActorTypePerson,
		vocab.ActorTypeService:

		if result.Outbox().IsMap() {
			cacheControl.MaxAge = clamp(hour, cacheControl.MaxAge, day)

		} else {
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
