package service

import (
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/davecgh/go-spew/spew"
)

type ActivityStreams struct {
	collection  data.Collection
	innerClient streams.Client
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func NewActivityStreams() ActivityStreams {
	return ActivityStreams{
		innerClient: streams.NewDefaultClient(),
	}
}

func (as *ActivityStreams) Refresh(innerClient streams.Client, collection data.Collection) {
	as.innerClient = innerClient
	as.collection = collection
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

func (as *ActivityStreams) Load(uri string) (streams.Document, error) {
	return as.innerClient.Load(uri)
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (client *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, maxRows int) (streams.Document, error) {
	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndLessThan("published", maxDate)

	results, err := client.query(criteria, option.SortDesc("published"), option.MaxRows(int64(maxRows)))

	return streams.NewDocument(results.Reverse(), streams.WithClient(client)),
		derp.Wrap(err, "emissary.tools.cache.Client.QueryRepliesAfterDate", "Error querying database")
}

func (client *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, maxRows int) (streams.Document, error) {

	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndGreaterThan("published", minDate)

	results, err := client.query(criteria, option.SortAsc("published"), option.MaxRows(int64(maxRows)))

	return streams.NewDocument(results, streams.WithClient(client)),
		derp.Wrap(err, "emissary.tools.cache.Client.QueryRepliesAfterDate", "Error querying database")
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (client *ActivityStreams) iterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return client.collection.List(criteria, options...)
}

// query reads from the database and returns a slice of streams.Document values
func (client *ActivityStreams) query(criteria exp.Expression, options ...option.Option) (sliceof.Object[mapof.Any], error) {

	if client.collection == nil {
		return make(sliceof.Object[mapof.Any], 0), nil
	}

	spew.Dump("query ---------------", criteria, options)
	iterator, err := client.iterator(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "emissary.tools.cache.Client.Query", "Error querying database")
	}

	result := make(sliceof.Object[mapof.Any], 0, iterator.Count())

	value := ascache.NewCachedValue()
	for iterator.Next(&value) {
		spew.Dump(value.Original)
		result = append(result, value.Original)
		value = ascache.NewCachedValue()

		if err := iterator.Error(); err != nil {
			return nil, derp.Wrap(err, "emisary.tools.cache.Client.Query", "Error during iteration")
		}
	}

	return result, nil
}
