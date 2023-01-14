package render

import (
	"io"
	"time"

	"github.com/benpate/derp"
)

// StepViewHTML represents an action-step that can render a Stream into HTML
type StepViewHTML struct {
	File string
}

// Get renders the Stream HTML to the context
func (step StepViewHTML) Get(renderer Renderer, buffer io.Writer) error {

	context := renderer.context()

	/* TODO: Re-implement this later.
	Caching leads to problems on INDEX-ONLY pages because you may have added/changed/deleted a child
	object, but the parent page is still cached.  So, you need to invalidate the cache for the parent.

	requestHeader := context.Request().Header

	// Validate If-None-Match Header
	if etag := requestHeader.Get("If-None-Match"); etag != "" {
		if etag == renderer.object().ETag() {
			context.Response().WriteHeader(http.StatusNotModified)
			return nil
		}
	}

	// Validate If-Modified-Since Header
	if modifiedSince := requestHeader.Get("If-Modified-Since"); modifiedSince != "" {
		if modifiedSinceDate, err := time.Parse(time.RFC3339, modifiedSince); err == nil {
			if modifiedSinceDate.UnixMilli() >= renderer.object().Updated() {
				context.Response().WriteHeader(http.StatusNotModified)
				return nil
			}
		}
	}
	*/

	header := context.Response().Header()

	header.Set("Vary", "Cookie, HX-Request")
	header.Set("Cache-Control", "private")

	// TODO: LOW: We can do a better job with caching.  If a page is public, then caching should be public, too.

	var filename string

	if step.File != "" {
		filename = step.File
	} else {
		filename = renderer.ActionID()
	}

	// TODO: MEDIUM: Re-implement caching.  Will need to automatically compute the "Vary" header.
	object := renderer.object()
	header.Set("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339))
	header.Set("ETag", object.ETag())

	if err := renderer.executeTemplate(buffer, filename, renderer); err != nil {
		return derp.Wrap(err, "render.StepViewHTML.Get", "Error executing template")
	}

	return nil
}

func (step StepViewHTML) UseGlobalWrapper() bool {
	return true
}

func (step StepViewHTML) Post(renderer Renderer) error {
	return nil
}
