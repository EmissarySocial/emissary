package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
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

func (client *ActivityStreams) QueryRepliesBeforeDate(inReplyTo string, maxDate int64, maxRows int64) (streams.Document, error) {
	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndLessOrEqual("published", maxDate)

	return client.query(criteria, option.SortDesc("published"), option.MaxRows(maxRows))
}

func (client *ActivityStreams) QueryRepliesAfterDate(inReplyTo string, minDate int64, maxRows int64) (streams.Document, error) {
	criteria := exp.
		Equal("inReplyTo", inReplyTo).
		AndGreaterOrEqual("published", minDate)

	return client.query(criteria, option.SortDesc("published"), option.MaxRows(maxRows))
}

/******************************************
 * Internal Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (client *ActivityStreams) iterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return client.collection.List(criteria, options...)
}

// query reads from the database and returns a slice of streams.Document values
func (client *ActivityStreams) query(criteria exp.Expression, options ...option.Option) (streams.Document, error) {

	if client.collection == nil {
		return streams.NilDocument(), nil
	}

	iterator, err := client.iterator(criteria, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, "emissary.tools.cache.Client.Query", "Error querying database")
	}

	result := make([]mapof.Any, 0, iterator.Count())

	value := mapof.NewAny()
	for iterator.Next(&value) {

		if err := iterator.Error(); err != nil {
			return streams.NilDocument(), derp.Wrap(err, "emisary.tools.cache.Client.Query", "Error during iteration")
		}

		result = append(result, value)
		value = mapof.NewAny()
	}

	return streams.NewDocument(result, streams.WithClient(client)), nil
}
