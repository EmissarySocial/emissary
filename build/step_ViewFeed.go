package build

import (
	"cmp"
	"encoding/json"
	"io"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/slice"
	"github.com/gorilla/feeds"
	"github.com/kr/jsonfeed"
	accept "github.com/timewasted/go-accept-headers"
)

// StepViewFeed is a Step that writes a Stream's children as an RSS, Atom, or JSON feed
type StepViewFeed struct {
	SearchTypes []string
}

// Get writes the feed in the format that the request asks for
func (step StepViewFeed) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepViewFeed.Get"

	mimeType := step.detectMimeType(builder)

	// Initialize the result RSS feed
	result := feeds.Feed{
		Title:       builder.Hostname() + " - " + builder.PageTitle(),
		Description: builder.Summary(),
		Link:        &feeds.Link{Href: builder.Permalink()},
		Author:      &feeds.Author{Name: ""},
		Created:     time.Now(),
	}

	switch len(step.SearchTypes) > 0 {

	// This is NOT a search builder
	case false:

		// Get the child streams that the CURRENT VIEWER is permitted to see.  This mirrors
		// the HTML/collection views (build.Stream.Children): the feed applies the same
		// defaultAllow permission filter, so gated (circle/paid/non-anonymous) children
		// are never leaked through the feed.
		criteria := exp.Equal("parentId", builder.objectID()).
			And(builder.defaultAllowed())

		children, err := builder.factory().Stream().List(builder.session(), criteria, feedListOptions()...)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Querying child streams"))
		}

		streams := feedOrder(iterator.Slice(children, model.NewStream))

		// Special case for JSONFeed
		if mimeType == model.MimeTypeJSONFeed {
			return step.asJSONFeed(builder, buffer, streams)
		}

		result.Items = slice.Map(streams, convert.StreamToGorillaFeed)

	// This IS a search builder
	case true:

		queryResults, err := builder.Search().
			Top120().
			ByCreateDate().
			Reverse().
			WhereType(step.SearchTypes...).
			Slice()

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Querying search queryResults"))
		}

		result.Items = slice.Map(queryResults, convert.SearchResultToGorillaFeed)
	}

	// Now write the feed straight into the buffer, in the requested format
	{
		var err error

		// Thank you gorilla/feeds for this awesome API.
		switch mimeType {

		case model.MimeTypeAtom:
			mimeType = "application/atom+xml; charset=UTF-8"
			err = result.WriteAtom(buffer)

		default:
			mimeType = "application/rss+xml; charset=UTF-8"
			err = result.WriteRss(buffer)
		}

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Writing feed to buffer"))
		}

		return Halt().AsFullPage().WithContentType(mimeType)
	}
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepViewFeed) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// detectMimeType chooses the feed format, preferring the "format" query parameter over the Accept header
func (step StepViewFeed) detectMimeType(builder Builder) string {

	// First, try to get the format from the query string
	switch builder.QueryParam("format") {

	case "json":
		return model.MimeTypeJSONFeed

	case "atom":
		return model.MimeTypeAtom

	case "rss":
		return model.MimeTypeRSS
	}

	// Otherwise, get the format from the "Accept" header
	header := builder.request().Header

	if result, err := accept.Negotiate(header.Get("Accept"), model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText); err == nil {
		return result
	}

	// Finally, use RSS as the default
	return model.MimeTypeRSS
}

// feedListOptions returns the query options for a feed's child streams: the feedMaxItems
// created most recently, loading only the fields the feed converters read
func feedListOptions() []option.Option {

	// RULE: Every Stream has a create date, and a page that was never sent to followers has
	// no publish date, so the newest are chosen by "createDate" and put in order by feedOrder
	return []option.Option{
		option.SortDesc("createDate"),
		option.MaxRows(feedMaxItems),
		option.Fields(convert.StreamFeedFields()...),
	}
}

// feedOrder sorts a feed's Streams by the date each one shows, newest first
func feedOrder(streams []model.Stream) []model.Stream {

	slices.SortStableFunc(streams, func(a model.Stream, b model.Stream) int {
		return cmp.Compare(b.DisplayDate(), a.DisplayDate())
	})

	return streams
}

// asJSONFeed renders the provided children as a JSONFeed document
func (step StepViewFeed) asJSONFeed(builder Builder, buffer io.Writer, children []model.Stream) PipelineBehavior {

	const location = "build.StepViewFeed.asJSONFeed"

	feed := jsonfeed.Feed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       builder.PageTitle(),
		HomePageURL: builder.Permalink(),
		FeedURL:     builder.Permalink() + "/feed?format=json",
		Description: builder.Summary(),
	}

	feed.Items = slice.Map(children, convert.StreamToJsonFeed)

	builder.response().Header().Add("Content-Type", model.MimeTypeJSONFeed)

	if err := json.NewEncoder(buffer).Encode(feed); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Writing JSONFeed to buffer"))
	}

	// Set ContentType
	return Halt().AsFullPage().WithContentType(model.MimeTypeJSONFeed)
}
