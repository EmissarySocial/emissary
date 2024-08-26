package build

import (
	"io"
)

// StepCacheURL is an action that can add new model objects of any type
type StepCacheURL struct {
	CacheControl string
}

// Get displays a modal form that lets users enter data for their new model object.
func (step StepCacheURL) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	header := builder.response().Header()

	// Handle Etag caching (if possible)
	if etag := builder.object().ETag(); etag != "" {
		if ifNoneMatch := builder.request().Header.Get("If-None-Match"); ifNoneMatch == etag {
			builder.response().WriteHeader(304)
			return Halt()
		}

		// Write Etag header
		header.Set("Etag", etag)
	}

	// Write cache control header
	if step.CacheControl != "" {
		header.Set("Cache-Control", step.CacheControl)
	}

	// TODO: Add in-memory caching on the server

	return nil
}

// Post initializes a new model object, populates it with data from the form, then saves it to the database.
func (step StepCacheURL) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}
