package render

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

// StepViewFeed represents an action-step that can render a Stream into HTML
type StepViewFeed struct{}

// Get renders the Stream HTML to the context
func (step StepViewFeed) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	const location = "render.StepViewFeed.Get"

	factory := renderer.factory()

	// Get all child streams from the database
	children, err := factory.Stream().ListByParent(renderer.objectID())

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Error querying child streams"))
	}

	mimeType := step.detectMimeType(renderer)

	// Special case for JSONFeed
	if mimeType == model.MimeTypeJSONFeed {
		return step.asJSONFeed(renderer, buffer, children)
	}

	// Initialize the result RSS feed
	result := feeds.Feed{
		Title:       renderer.PageTitle(),
		Description: renderer.Summary(),
		Link:        &feeds.Link{Href: renderer.Permalink()},
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
			return ExitError(derp.Wrap(err, location, "Error generating feed. This should never happen"))
		}

		// nolint:errcheck
		buffer.Write([]byte(xml))
		return ExitHalt().WithContentType(mimeType)
	}
}

func (step StepViewFeed) Post(renderer Renderer, _ io.Writer) ExitCondition {
	return nil
}

func (step StepViewFeed) detectMimeType(renderer Renderer) string {

	context := renderer.context()

	// First, try to get the format from the query string
	switch context.QueryParam("format") {
	case "json":
		return model.MimeTypeJSONFeed
	case "atom":
		return model.MimeTypeAtom
	case "rss":
		return model.MimeTypeRSS
	}

	// Otherwise, get the format from the "Accept" header
	header := context.Request().Header

	if result, err := accept.Negotiate(header.Get("Accept"), model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText); err == nil {
		return result
	}

	// Finally, use JSONFeed as the default
	return model.MimeTypeJSONFeed
}

func (step StepViewFeed) asJSONFeed(renderer Renderer, buffer io.Writer, children data.Iterator) ExitCondition {

	context := renderer.context()

	feed := jsonfeed.Feed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       renderer.PageTitle(),
		HomePageURL: renderer.Permalink(),
		FeedURL:     renderer.Permalink() + "/feed?format=json",
		Description: renderer.Summary(),
		Hubs: []jsonfeed.Hub{
			{
				Type: "WebSub",
				URL:  renderer.Permalink() + "/websub",
			},
		},
	}

	feed.Items = slice.Map(iterator.Slice(children, model.NewStream), convert.StreamToJsonFeed)

	context.Response().Header().Add("Content-Type", model.MimeTypeJSONFeed)

	bytes, err := json.Marshal(feed)

	if err != nil {
		return ExitError(derp.Wrap(err, "render.StepViewFeed.asJSONFeed", "Error generating JSONFeed"))
	}

	// nolint:errcheck
	buffer.Write(bytes)

	// Set ContentType
	return ExitHalt().WithContentType(model.MimeTypeJSONFeed)
}
