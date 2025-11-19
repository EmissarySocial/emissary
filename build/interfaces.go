package build

import (
	"github.com/benpate/form"
)

type PipelineHalter interface {

	// HaltPipeline optionally allows a step to halt processing of an action pipeline
	HaltPipeline(Builder) bool
}

type StateSetter interface {
	setState(string) error
}

type PropertyFormGetter interface {
	PropertyForm() form.Element
}
