package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/html"
)

// StepProcessContent is an action step that adds tags to a stream, either by scanning the content, or by
// calculating template values
type StepProcessContent struct {
	RemoveHTML bool
	AddTags    bool
	AddLinks   bool
}

// Get renders the HTML for this step - either a modal template selector, or the embedded edit form
func (step StepProcessContent) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepProcessContent) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {

	const location = "render.StepProcessContent.Post"

	// Require that we are working with a Stream object
	streamRenderer, ok := renderer.(*Stream)

	if !ok {
		return Halt().WithError(derp.NewInternalError(location, "step: AddTags can only be used on a Stream"))
	}

	factory := streamRenderer.factory()
	streamService := factory.Stream()
	contentService := factory.Content()

	stream := streamRenderer._stream

	if step.RemoveHTML {
		stream.Content.HTML = html.RemoveTags(stream.Content.HTML)
	}

	if step.AddLinks {
		contentService.ApplyLinks(&stream.Content)
	}

	if step.AddTags {
		streamService.CalcTags(stream)
		contentService.ApplyTags(&stream.Content, stream.Tags)
	}

	return Continue()
}
