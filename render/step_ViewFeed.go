package render

import (
	"io"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/gorilla/feeds"
	"github.com/kr/jsonfeed"
	accept "github.com/timewasted/go-accept-headers"
)

// StepViewFeed represents an action-step that can render a Stream into HTML
type StepViewFeed struct{}

// Get renders the Stream HTML to the context
func (step StepViewFeed) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepViewFeed.Get"

	factory := renderer.factory()

	// Get all child streams from the database
	children, err := factory.Stream().ListByParent(renderer.objectID())

	if err != nil {
		return derp.Wrap(err, location, "Error querying child streams")
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

	result.Items = []*feeds.Item{}

	// Iterate through list of children and add to the RSS feed
	stream := model.NewStream()

	for children.Next(&stream) {
		result.Items = append(result.Items, &feeds.Item{
			Title:       stream.Document.Label,
			Description: stream.Document.Summary,
			Link: &feeds.Link{
				Href: stream.Document.URL,
			},
			Author: &feeds.Author{
				Name:  stream.Document.Author.Name,
				Email: stream.Document.Author.EmailAddress,
			},
			Created: time.UnixMilli(stream.PublishDate),
		})

		stream = model.NewStream() // Reset the stream variable so we don't get collisions
	}

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
			return derp.Wrap(err, location, "Error generating feed. This should never happen")
		}

		// Write the result to the buffer and then success.
		header := renderer.context().Response().Header()
		header.Add("Content-Type", mimeType)
		buffer.Write([]byte(xml))
		return nil
	}
}

func (step StepViewFeed) UseGlobalWrapper() bool {
	return false
}

func (step StepViewFeed) Post(renderer Renderer) error {
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

	if result, err := accept.Negotiate(header.Get("Accept"), model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS); err == nil {
		return result
	}

	// Finally, use JSONFeed as the default
	return model.MimeTypeJSONFeed
}

func (step StepViewFeed) asJSONFeed(renderer Renderer, buffer io.Writer, children data.Iterator) error {

	context := renderer.context()

	feed := jsonfeed.Feed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       renderer.PageTitle(),
		HomePageURL: renderer.Permalink(),
		FeedURL:     "",
		Description: renderer.Summary(),
		Hubs: []jsonfeed.Hub{
			{
				Type: "WebSub",
				URL:  renderer.Permalink() + "/websub",
			},
		},
	}

	stream := model.NewStream()
	for children.Next(&stream) {
		feed.Items = append(feed.Items, jsonfeed.Item{
			ID:            stream.Token,
			URL:           stream.Document.URL,
			Title:         stream.Document.Label,
			ContentHTML:   stream.Content.HTML,
			Summary:       stream.Document.Summary,
			Image:         stream.Document.ImageURL,
			DatePublished: time.UnixMilli(stream.PublishDate),
			DateModified:  time.UnixMilli(stream.UpdateDate),
			Author: &jsonfeed.Author{
				Name:   stream.Document.Author.Name,
				URL:    stream.Document.Author.ProfileURL,
				Avatar: stream.Document.Author.ImageURL,
			},
			// TODO: Attachments for podcasts, etc.
		})

		stream = model.NewStream() // Reset the variable to prevent collisions
	}

	context.Response().Header().Add("Content-Type", model.MimeTypeJSONFeed)

	return context.JSON(http.StatusOK, feed)
}
