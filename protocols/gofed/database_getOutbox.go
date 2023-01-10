package gofed

import (
	"context"
	"net/url"
	"strconv"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/protocols/gofed/as"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	builder "github.com/benpate/exp-builder"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// GetOutbox returns the latest page of the inbox corresponding to the outboxIRI.
//
// It is similar in behavior to its GetInbox counterpart, but for the actor's Outbox
// instead. See the similar documentation for GetInbox.
func (db Database) GetOutbox(c context.Context, outboxIRI *url.URL) (inbox vocab.ActivityStreamsOrderedCollectionPage, err error) {

	const location = "gofed.Database.GetOutbox"

	// Parse the URL to get the OwnerID
	ownerID, _, _, err := ParsePath(outboxIRI)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing URL", outboxIRI.String())
	}

	// Criteria Builder for Pagination
	builder := builder.NewBuilder().
		Int("document.publishDate")

	criteria := builder.Evaluate(outboxIRI.Query())

	// Query the database
	it, err := db.activityService.ListOutbox(ownerID, criteria, option.MaxRows(60))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	// Build the list of items
	items := streams.NewActivityStreamsOrderedItemsProperty()
	activity := model.NewActivity()
	for it.Next(&activity) {
		if record, err := ToGoFed(&activity); err == nil {
			items.AppendType(record)
		} else {
			derp.Report(derp.Wrap(err, location, "Error serializing activity", activity))
		}
		activity = model.NewActivity()
	}

	if err := it.Error(); err != nil {
		return nil, derp.Wrap(err, location, "Error iterating over result set")
	}

	// Build the OrderedCollectionPage
	result := streams.NewActivityStreamsOrderedCollectionPage()
	result.SetActivityStreamsOrderedItems(items)

	// Add "Next Page" link (if more than zero results)
	if !activity.IsNew() {

		nextPageURL, _ := url.Parse(outboxIRI.String())
		nextPageURL.RawQuery = "document.publishDate=LT:" + strconv.FormatInt(activity.Document.PublishDate, 10)

		nextPage := streams.NewActivityStreamsNextProperty()
		as.SetLink(nextPage, nextPageURL, "Next Page", "Link")

		result.SetActivityStreamsNext(nextPage)
	}

	return result, nil
}
