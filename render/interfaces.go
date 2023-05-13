package render

import "github.com/EmissarySocial/emissary/model"

type PipelineHalter interface {

	// HaltPipeline optionally allows a step to halt processing of an action pipeline
	HaltPipeline(Renderer) bool
}

type DocumentLinker interface {
	DocumentLink() model.DocumentLink
}
