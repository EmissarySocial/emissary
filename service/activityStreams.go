package service

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// ActivityStreams implements the Hannibal HTTP client interface, and provides a cache for ActivityStreams documents.
type ActivityStreams struct {
	documentCollection data.Collection
	innerClient        streams.Client
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewActivityStreams creates a new ActivityStreams service
func NewActivityStreams() ActivityStreams {
	return ActivityStreams{}
}

// Refresh updates the ActivityStreams service with new dependencies
func (service *ActivityStreams) Refresh(innerClient streams.Client, documentCollection data.Collection) {
	service.innerClient = innerClient
	service.documentCollection = documentCollection
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

// Load implements the Hannibal `Client` interface, and returns a streams.Document from the cache.
func (service *ActivityStreams) Load(uri string, options ...any) (streams.Document, error) {

	// NPE Check
	if service.innerClient == nil {
		return streams.Document{}, derp.NewInternalError("service.ActivityStreams.Load", "Client not initialized")
	}

	// Forward request to inner client
	return service.innerClient.Load(uri, options...)
}

/******************************************
 * Custom Behaviors
 ******************************************/

// PurgeCache removes all expired documents from the cache
func (service *ActivityStreams) PurgeCache() error {

	// NPE Check
	if service.documentCollection == nil {
		return derp.NewInternalError("service.ActivityStreams.PurgeCache", "Document Collection not initialized")
	}

	criteria := exp.LessThan("expires", time.Now().Unix())

	// Purge all expired Documents
	if err := service.documentCollection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.ActivityStreams.PurgeCache", "Error purging documents")
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

// DeleteDocumentByURL removes a single document from the database by its URL
func (service *ActivityStreams) DeleteDocumentByURL(url string) error {

	// NPE Check
	if service.documentCollection == nil {
		return derp.NewInternalError("service.ActivityStreams.DeleteDocumentByURL", "Document Collection not initialized")
	}

	// Forward request to documentCollection
	return service.documentCollection.HardDelete(exp.Equal("uri", url))
}

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, maxRows int) (streams.Document, error) {

	// NPE Check
	if service.documentCollection == nil {
		return streams.Document{}, derp.NewInternalError("service.ActivityStreams.QueryRepliesBeforeDate", "Document Collection not initialized")
	}

	// Build the query
	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndLessThan("published", maxDate)

	results, err := service.documentQuery(criteria, option.SortDesc("published"), option.MaxRows(int64(maxRows)))

	// Return the results as a streams.Document / collection
	return streams.NewDocument(results.Reverse(), streams.WithClient(service)),
		derp.Wrap(err, "service.ActivityStreams.QueryRepliesAfterDate", "Error querying database")
}

// QueryRepliesAfterDate returns a slice of streams.Document values that are replies to the specified document, and were published after the specified date.
func (service *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, maxRows int) (streams.Document, error) {

	// NPE Check
	if service.documentCollection == nil {
		return streams.Document{}, derp.NewInternalError("service.ActivityStreams.QueryRepliesAfterDate", "Document Collection not initialized")
	}

	// Build the query
	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndGreaterThan("published", minDate)

	results, err := service.documentQuery(criteria, option.SortAsc("published"), option.MaxRows(int64(maxRows)))

	// Return the result as a streams.Document / collection
	return streams.NewDocument(results, streams.WithClient(service)),
		derp.Wrap(err, "service.ActivityStreams.QueryRepliesAfterDate", "Error querying database")
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (service *ActivityStreams) documentIterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {

	// NPE Check
	if service.documentCollection == nil {
		return nil, derp.NewInternalError("service.ActivityStreams.documentIterator", "Document Collection not initialized")
	}

	// Forward request to documentCollection
	return service.documentCollection.Iterator(criteria, options...)
}

// query reads from the database and returns a slice of streams.Document values
func (service *ActivityStreams) documentQuery(criteria exp.Expression, options ...option.Option) (sliceof.Object[mapof.Any], error) {

	// NPE Check
	if service.documentCollection == nil {
		return nil, derp.NewInternalError("service.ActivityStreams.documentQuery", "Document Collection not initialized")
	}

	// Create the Iterator
	iterator, err := service.documentIterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.ActivityStreams.Query", "Error querying database")
	}

	// Initialize result slice
	result := make(sliceof.Object[mapof.Any], 0, iterator.Count())

	// Map iterator into results
	value := ascache.NewCachedValue()
	for iterator.Next(&value) {
		result = append(result, value.Original)
		value = ascache.NewCachedValue()

		if err := iterator.Error(); err != nil {
			return nil, derp.Wrap(err, "emisary.tools.cache.Client.Query", "Error during iteration")
		}
	}

	// Return success
	return result, nil
}
