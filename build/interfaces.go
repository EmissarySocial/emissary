package build

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
)

type PipelineHalter interface {

	// HaltPipeline optionally allows a step to halt processing of an action pipeline
	HaltPipeline(Builder) bool
}

type DocumentLinker interface {
	DocumentLink() model.DocumentLink
}

type StateSetter interface {
	setState(string) error
}

type PropertyFormGetter interface {
	PropertyForm() form.Element
}

type SearchResulter interface {
	SearchResult() model.SearchResult
}
