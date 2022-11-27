package render

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/dyatlov/go-htmlinfo/htmlinfo"
)

// ExpandURL represents an action-step that can update the custom data stored in a Stream
type ExpandURL struct {
	Path string
}

func (step ExpandURL) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step ExpandURL) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step ExpandURL) Post(renderer Renderer) error {

	const location = "render.ExpandURL.Post"

	// Get the stream renderer
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream
	targetURL := step.getURL(stream)

	// If there is no origin URL, then there is nothing to do.
	if targetURL == "" {
		return nil
	}

	// Load the original URL
	var body bytes.Buffer
	txn := remote.Get(targetURL).Response(&body, nil)

	if err := txn.Send(); err != nil {
		derp.Report(derp.Wrap(err, location, "Error fetching remote URL", targetURL))
	}

	// Parse the response
	contentType := txn.ResponseObject.Header.Get("Content-Type")
	info := htmlinfo.NewHTMLInfo()

	if err := info.Parse(&body, &targetURL, &contentType); err != nil {
		derp.Report(derp.Wrap(err, location, "Error parsing remote URL", targetURL))
		return nil
	}

	// Set data in the stream
	switch step.Path {
	case "origin":
		stream.Origin.URL = info.CanonicalURL
		stream.Origin.Label = info.Title
		stream.Origin.Summary = info.Description
		stream.Origin.ImageURL = info.ImageSrcURL

	case "document":
		stream.Document.URL = info.CanonicalURL
		stream.Document.Label = info.Title
		stream.Document.Summary = info.Description
		stream.Document.ImageURL = info.ImageSrcURL
	}

	// Silence is AU-some
	return nil
}

func (step ExpandURL) getURL(stream *model.Stream) string {

	switch step.Path {
	case "origin":
		return stream.Origin.URL

	case "document":
		return stream.Document.URL

	default:
		return ""
	}
}
