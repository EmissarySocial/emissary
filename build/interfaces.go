package build

import (
	"github.com/benpate/form"
)

// PipelineHalter is implemented by a Step that can stop the rest of the pipeline from running
type PipelineHalter interface {

	// HaltPipeline optionally allows a step to halt processing of an action pipeline
	HaltPipeline(Builder) bool
}

// StateSetter is implemented by a Builder whose object can be moved into a new state
type StateSetter interface {
	setState(string) error
}

// PropertyFormGetter is implemented by a Builder that exposes a settings form
type PropertyFormGetter interface {
	PropertyForm() form.Element
}
