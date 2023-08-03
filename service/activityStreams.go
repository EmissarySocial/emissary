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
	innerClient        streams.Client
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func NewActivityStreams() ActivityStreams {
	return ActivityStreams{
		innerClient: streams.NewDefaultClient(),
	}
}

func (as *ActivityStreams) Refresh(innerClient streams.Client, actorCollection data.Collection, documentCollection data.Collection) {
	as.innerClient = innerClient
	as.actorCollection = actorCollection
	as.documentCollection = documentCollection
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

func (as *ActivityStreams) LoadDocument(uri string, defaultValue map[string]any) (streams.Document, error) {
	return as.innerClient.LoadDocument(uri, defaultValue)
}

func (as *ActivityStreams) LoadActor(uri string) (streams.Document, error) {
	return as.innerClient.LoadActor(uri)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (client *ActivityStreams) PurgeCache() error {
	criteria := exp.LessThan("expires", time.Now().Unix())

	// Purge all expired Actors
	// if err := client.actorCollection.HardDelete(criteria); err != nil {
	//	return derp.Wrap(err, "emissary.tools.cache.Client.PurgeCache", "Error purging actors")
	// }

	// Purge all expired Documents
	if err := client.documentCollection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "emissary.tools.cache.Client.PurgeCache", "Error purging documents")
	}

	return nil
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (client *ActivityStreams) DeleteDocumentByURL(url string) error {
	return client.documentCollection.HardDelete(exp.Equal("uri", url))
}

func (client *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, maxRows int) (streams.Document, error) {
	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndLessThan("published", maxDate)

	results, err := client.documentQuery(criteria, option.SortDesc("published"), option.MaxRows(int64(maxRows)))

	return streams.NewDocument(results.Reverse(), streams.WithClient(client)),
		derp.Wrap(err, "emissary.tools.cache.Client.QueryRepliesAfterDate", "Error querying database")
}

func (client *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, maxRows int) (streams.Document, error) {

	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndGreaterThan("published", minDate)

	results, err := client.documentQuery(criteria, option.SortAsc("published"), option.MaxRows(int64(maxRows)))

	return streams.NewDocument(results, streams.WithClient(client)),
		derp.Wrap(err, "emissary.tools.cache.Client.QueryRepliesAfterDate", "Error querying database")
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (client *ActivityStreams) documentIterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return client.documentCollection.List(criteria, options...)
}

// query reads from the database and returns a slice of streams.Document values
func (client *ActivityStreams) documentQuery(criteria exp.Expression, options ...option.Option) (sliceof.Object[mapof.Any], error) {

	if client.documentCollection == nil {
		return make(sliceof.Object[mapof.Any], 0), nil
	}

	iterator, err := client.documentIterator(criteria, options...)

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
