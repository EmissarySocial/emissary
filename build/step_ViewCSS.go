package build

import (
	"io"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/compare"
)

// StepViewCSS is a Step that can build a Stream into HTML
type StepViewCSS struct {
	File string
}

// Get builds the Stream HTML to the context
func (step StepViewCSS) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	/* TODO: MEDIUM: Re-implement client-side caching later.
	Caching leads to problems on INDEX-ONLY pages because you may have added/changed/deleted a child
	object, but the parent page is still cached.  So, you need to invalidate the cache for the parent.

	requestHeader := context.Request().Header

	// Validate If-None-Match Header
	if etag := requestHeader.Get("If-None-Match"); etag != "" {
		if etag == builder.object().ETag() {
			context.Response().WriteHeader(http.StatusNotModified)
			return nil
		}
	}

	// Validate If-Modified-Since Header
	if modifiedSince := requestHeader.Get("If-Modified-Since"); modifiedSince != "" {
		if modifiedSinceDate, err := time.Parse(time.RFC3339, modifiedSince); err == nil {
			if modifiedSinceDate.UnixMilli() >= builder.object().Updated() {
				context.Response().WriteHeader(http.StatusNotModified)
				return nil
			}
		}
	}
	*/

	var filename string

	if step.File != "" {
		filename = step.File
	} else {
		filename = builder.actionID()
	}

	if err := builder.execute(buffer, filename, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepViewCSS.Get", "Error executing template"))
	}

	// TODO: MEDIUM: Re-implement caching.  Will need to automatically compute the "Vary" header.
	result := Halt().
		AsFullPage().
		WithHeader("Content-Type", "text/css").
		WithHeader("Vary", "Cookie, HX-Request")

	// If we have a valid object, then try to set ETag headers.
	if object := builder.object(); compare.NotNil(object) {
		result = result.
			WithHeader("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339)).
			WithHeader("ETag", object.ETag())
	}

	// Otherwise, just continue without headers.
	return result
}

func (step StepViewCSS) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}
