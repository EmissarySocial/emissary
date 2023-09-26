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

type ActivityStreams struct {
	actorCollection    data.Collection
	documentCollection data.Collection
	innerClient        *ascache.Client
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func NewActivityStreams() ActivityStreams {
	return ActivityStreams{}
}

func (service *ActivityStreams) Refresh(innerClient *ascache.Client, actorCollection data.Collection, documentCollection data.Collection) {
	service.innerClient = innerClient
	service.actorCollection = actorCollection
	service.documentCollection = documentCollection
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

func (service *ActivityStreams) Load(uri string, options ...any) (streams.Document, error) {
	return service.innerClient.Load(uri, options...)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *ActivityStreams) PurgeCache() error {
	criteria := exp.LessThan("expires", time.Now().Unix())

	// Purge all expired Actors
	// if err := service.actorCollection.HardDelete(criteria); err != nil {
	//	return derp.Wrap(err, "emissary.tools.cache.Client.PurgeCache", "Error purging actors")
	// }

	// Purge all expired Documents
	if err := service.documentCollection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "emissary.tools.cache.Client.PurgeCache", "Error purging documents")
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *ActivityStreams) DeleteDocumentByURL(url string) error {
	return service.documentCollection.HardDelete(exp.Equal("uri", url))
}

func (service *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, maxRows int) (streams.Document, error) {
	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndLessThan("published", maxDate)

	results, err := service.documentQuery(criteria, option.SortDesc("published"), option.MaxRows(int64(maxRows)))

	return streams.NewDocument(results.Reverse(), streams.WithClient(service)),
		derp.Wrap(err, "emissary.tools.cache.Client.QueryRepliesAfterDate", "Error querying database")
}

func (service *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, maxRows int) (streams.Document, error) {

	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndGreaterThan("published", minDate)

	results, err := service.documentQuery(criteria, option.SortAsc("published"), option.MaxRows(int64(maxRows)))

	return streams.NewDocument(results, streams.WithClient(service)),
		derp.Wrap(err, "emissary.tools.cache.Client.QueryRepliesAfterDate", "Error querying database")
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (service *ActivityStreams) documentIterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.documentCollection.Iterator(criteria, options...)
}

// query reads from the database and returns a slice of streams.Document values
func (service *ActivityStreams) documentQuery(criteria exp.Expression, options ...option.Option) (sliceof.Object[mapof.Any], error) {

	if service.documentCollection == nil {
		return make(sliceof.Object[mapof.Any], 0), nil
	}

	iterator, err := service.documentIterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "emissary.tools.cache.Client.Query", "Error querying database")
	}

	result := make(sliceof.Object[mapof.Any], 0, iterator.Count())

	value := ascache.NewCachedValue()
	for iterator.Next(&value) {
		result = append(result, value.Original)
		value = ascache.NewCachedValue()

		if err := iterator.Error(); err != nil {
			return nil, derp.Wrap(err, "emisary.tools.cache.Client.Query", "Error during iteration")
		}
	}

	return result, nil
}
