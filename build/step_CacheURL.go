package build

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/rosetta/compare"
)

// StepCacheURL is an action that can add new model objects of any type
type StepCacheURL struct {
	CacheControl string
}

// Get answers a conditional request with a 304, or decorates the response with its cache headers.
func (step StepCacheURL) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	var etag string

	// RULE: Authenticated users and guest identities should not be served cached content. Skip this step for them.
	if builder.IsAuthenticatedOrIdentity() {
		return Continue()
	}

	// Read the entity-tag (if possible).  Not every Builder wraps a model object, so guard the
	// same way StepViewHTML does before reading one.
	if object := builder.object(); compare.NotNil(object) {

		// RULE: The tag is assembled exactly as StepViewHTML assembles it, because that step
		// publishes the tag the client keeps -- the pipeline applies its headers last.  Comparing a
		// bare revision against a variant-tagged one never matches, so the 304 below never fires.
		if object.ETag() != "" {
			etag = headers.ETag(headers.VariantHTML, object)
		}
	}

	// RULE: A 304 repeats the headers its 200 would have carried and has no body, so the
	// pipeline halts here rather than building a page that is thrown away.  AsFullPage is what
	// stops AsHTML from wrapping the empty body in the site theme and answering 200 instead.
	if (etag != "") && (builder.request().Header.Get("If-None-Match") == etag) {
		return step.withCacheHeaders(Halt().AsFullPage().WithStatusCode(http.StatusNotModified), etag)
	}

	return step.withCacheHeaders(Continue(), etag)
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepCacheURL) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// withCacheHeaders adds this step's cache headers to a behavior, so that the 304 and the 200
// answer with the same validator and the same policy.
func (step StepCacheURL) withCacheHeaders(behavior PipelineBehavior, etag string) PipelineBehavior {

	if etag != "" {
		behavior = behavior.WithHeader("ETag", etag)
	}

	if step.CacheControl != "" {
		behavior = behavior.WithHeader("Cache-Control", step.CacheControl)
	}

	return behavior
}
