package render

import (
	"io"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/compare"
)

// StepViewHTML represents an action-step that can render a Stream into HTML
type StepViewHTML struct {
	File   string
	Method string
}

// Get renders the Stream HTML to the context
func (step StepViewHTML) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	if step.Method != "post" {
		return step.execute(renderer, buffer)
	}

	return nil
}

func (step StepViewHTML) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {

	if step.Method != "get" {
		return step.execute(renderer, buffer)
	}

	return nil
}

func (step StepViewHTML) execute(renderer Renderer, buffer io.Writer) PipelineBehavior {

	/* TODO: MEDIUM: Re-implement client-side caching later.
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

	// TODO: LOW: We can do a better job with caching.  If a page is public, then caching should be public, too.
	header := renderer.response().Header()
	header.Set("Vary", "Cookie, HX-Request")
	// header.Set("Cache-Control", "private")

	var filename string

	if step.File != "" {
		filename = step.File
	} else {
		filename = renderer.ActionID()
	}

	if err := renderer.executeTemplate(buffer, filename, renderer); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepViewHTML.Get", "Error executing template"))
	}

	// TODO: MEDIUM: Re-implement caching.  Will need to automatically compute the "Vary" header.

	// If we have a valid object, then try to set ETag headers.
	if object := renderer.object(); compare.NotNil(object) {
		return Continue().
			WithHeader("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339)).
			WithHeader("ETag", object.ETag())
	}

	// Otherwise, just continue without headers.
	return Continue()
}
