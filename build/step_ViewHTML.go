package build

import (
	"io"
	"time"

	"github.com/EmissarySocial/emissary/tools/negotiate"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/compare"
)

// StepViewHTML is a Step that can build a Stream into HTML
type StepViewHTML struct {
	File       string
	Method     string
	AsFullPage bool
}

// Get builds the Stream HTML to the context
func (step StepViewHTML) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Method != "post" {
		return step.execute(builder, buffer)
	}

	return nil
}

// Post builds the Stream HTML to the context
func (step StepViewHTML) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Method != "get" {
		return step.execute(builder, buffer)
	}

	return nil
}

// execute renders the template and decorates the response with its cache-validation headers
func (step StepViewHTML) execute(builder Builder, buffer io.Writer) PipelineBehavior {

	// This step publishes ETag/Last-Modified below, but deliberately does NOT act on If-None-Match
	// or If-Modified-Since.  An INDEX-ONLY page goes stale as soon as a child is added, changed, or
	// deleted, and nothing invalidates the parent when that happens -- so a 304 here would serve a
	// listing that is missing its newest entry.

	// Cache-Control is left unset rather than pinned to "private": a public page should be publicly
	// cacheable, and this step cannot yet tell the two apart.
	header := builder.response().Header()

	// RULE: this must include `Accept` -- the same URL serves two representations, so a cache that
	// ignores it would hand a peer's AS2 document to a browser
	header.Set("Vary", negotiate.VaryHTML)

	// Render the named template, defaulting to the one that matches the action
	var filename string

	if step.File != "" {
		filename = step.File
	} else {
		filename = builder.actionID()
	}

	if err := builder.execute(buffer, filename, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepViewHTML.Get", "Executing template"))
	}

	// If we have a valid object, then try to set ETag headers.
	result := Continue()

	if object := builder.object(); compare.NotNil(object) {
		result = result.
			WithHeader("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339)).
			WithHeader("ETag", object.ETag())
	}

	// If "as-full-page" was specified, then include that in the result
	if step.AsFullPage {
		result = result.AsFullPage()
	}

	// Otherwise, just continue without headers.
	return result
}
