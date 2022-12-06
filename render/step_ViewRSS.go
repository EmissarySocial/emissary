package render

import (
	"io"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/gorilla/feeds"
)

// StepViewRSS represents an action-step that can render a Stream into HTML
type StepViewRSS struct {
	Format string // atom, rss, json (default is rss)
}

// Get renders the Stream HTML to the context
func (step StepViewRSS) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepViewRSS.Get"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)

	// Get all child streams from the database
	children, err := factory.Stream().ListByParent(renderer.objectID())

	if err != nil {
		return derp.Wrap(err, location, "Error querying child streams")
	}

	// Initialize the result RSS feed
	result := feeds.Feed{
		Title:       "",
		Description: "",
		Link:        &feeds.Link{Href: streamRenderer.Permalink()},
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
		var mimeType string
		var err error

		// Thank you gorilla/feeds for this awesome API.
		switch step.Format {
		case "atom":
			mimeType = "application/atom+xml; charset=UTF=8"
			xml, err = result.ToAtom()

		case "json":
			mimeType = "application/json; charset=UTF=8"
			xml, err = result.ToJSON()

		default:
			mimeType = "application/rss+xml; charset=UTF=8"
			xml, err = result.ToRss()
		}

		if err != nil {
			return derp.Wrap(err, location, "Error generating feed", step.Format)
		}

		// Write the result to the buffer and then success.
		header := renderer.context().Response().Header()
		header.Add("Content-Type", mimeType)
		buffer.Write([]byte(xml))
		return nil
	}
}

func (step StepViewRSS) UseGlobalWrapper() bool {
	return false
}

func (step StepViewRSS) Post(renderer Renderer) error {
	return nil
}
