package render

type PipelineHalter interface {

	// HaltPipeline optionally allows a step to halt processing of an action pipeline
	HaltPipeline(Renderer) bool
}
