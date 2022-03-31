package render

import (
	"io"
	"time"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/gorilla/feeds"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/service"
)

// StepViewRSS represents an action-step that can render a Stream into HTML
type StepViewRSS struct {
	streamService *service.Stream
	Format        string // atom, rss, json (default is rss)
}

// NewStepViewRSS generates a fully initialized StepViewRSS step.
func NewStepViewRSS(streamService *service.Stream, stepInfo datatype.Map) StepViewRSS {

	return StepViewRSS{
		streamService: streamService,
		Format:        convert.String(stepInfo["format"]),
	}
}

// Get renders the Stream HTML to the context
func (step StepViewRSS) Get(buffer io.Writer, renderer Renderer) error {

	const location = "render.StepViewRSS.Get"

	streamRenderer := renderer.(*Stream)

	// Get all child streams from the database
	children, err := step.streamService.ListByParent(renderer.objectID())

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
		permalink := renderer.Protocol() + renderer.Hostname() + "/" + stream.StreamID.Hex()
		result.Items = append(result.Items, &feeds.Item{
			Title:       stream.Label,
			Description: stream.Description,
			Link: &feeds.Link{
				Href: permalink,
			},
			Author: &feeds.Author{
				Name: stream.AuthorName,
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
			mimeType = "application/atom+xml"
			xml, err = result.ToAtom()

		case "json":
			mimeType = "application/json"
			xml, err = result.ToJSON()

		default:
			mimeType = "application/rss+xml"
			xml, err = result.ToRss()
		}

		if err != nil {
			return derp.Wrap(err, location, "Error generating feed", step.Format)
		}

		// Write the result to the buffer and then success.
		renderer.context().Response().Header().Add("mime-type", mimeType)
		buffer.Write([]byte(xml))
		return nil
	}
}

// Post is not supported for this step.
func (step StepViewRSS) Post(buffer io.Writer, renderer Renderer) error {
	return nil
}
