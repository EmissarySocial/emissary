package build

import (
	"encoding/json"
	"io"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/slice"
	"github.com/gorilla/feeds"
	"github.com/kr/jsonfeed"
	accept "github.com/timewasted/go-accept-headers"
)

// StepViewFeed represents an action-step that can build a Stream into HTML
type StepViewFeed struct{}

// Get builds the Stream HTML to the context
func (step StepViewFeed) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepViewFeed.Get"

	factory := builder.factory()

	// Get all child streams from the database
	children, err := factory.Stream().ListByParent(builder.objectID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error querying child streams"))
	}

	mimeType := step.detectMimeType(builder)

	// Special case for JSONFeed
	if mimeType == model.MimeTypeJSONFeed {
		return step.asJSONFeed(builder, buffer, children)
	}

	// Initialize the result RSS feed
	result := feeds.Feed{
		Title:       builder.PageTitle(),
		Description: builder.Summary(),
		Link:        &feeds.Link{Href: builder.Permalink()},
		Author:      &feeds.Author{Name: ""},
		Created:     time.Now(),
	}

	result.Items = slice.Map(iterator.Slice(children, model.NewStream), convert.StreamToGorillaFeed)

	// Now write the feed into the requested format
	{
		var xml string
		var err error

		// Thank you gorilla/feeds for this awesome API.
		switch mimeType {

		case model.MimeTypeAtom:
			mimeType = "application/atom+xml; charset=UTF=8"
			xml, err = result.ToAtom()

		case model.MimeTypeRSS:
			mimeType = "application/rss+xml; charset=UTF=8"
			xml, err = result.ToRss()
		}

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error generating feed. This should never happen"))
		}

		// nolint:errcheck
		buffer.Write([]byte(xml))
		return Halt().AsFullPage().WithContentType(mimeType)
	}
}

func (step StepViewFeed) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

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

	// Finally, use JSONFeed as the default
	return model.MimeTypeJSONFeed
}

func (step StepViewFeed) asJSONFeed(builder Builder, buffer io.Writer, children data.Iterator) PipelineBehavior {

	feed := jsonfeed.Feed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       builder.PageTitle(),
		HomePageURL: builder.Permalink(),
		FeedURL:     builder.Permalink() + "/feed?format=json",
		Description: builder.Summary(),
		Hubs: []jsonfeed.Hub{
			{
				Type: "WebSub",
				URL:  builder.Permalink() + "/websub",
			},
		},
	}

	feed.Items = slice.Map(iterator.Slice(children, model.NewStream), convert.StreamToJsonFeed)

	builder.response().Header().Add("Content-Type", model.MimeTypeJSONFeed)

	bytes, err := json.Marshal(feed)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepViewFeed.asJSONFeed", "Error generating JSONFeed"))
	}

	// nolint:errcheck
	buffer.Write(bytes)

	// Set ContentType
	return Halt().AsFullPage().WithContentType(model.MimeTypeJSONFeed)
}
