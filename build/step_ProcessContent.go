package build

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/html"
)

// StepProcessContent is an action step that adds tags to a stream, either by scanning the content, or by
// calculating template values
type StepProcessContent struct {
	Format     string
	RemoveHTML bool
	AddLinks   bool
	AddTags    bool
	TagPath    string
}

// Get builds the HTML for this step - either a modal template selector, or the embedded edit form
func (step StepProcessContent) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepProcessContent) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepProcessContent.Post"

	// Require that we are working with a Stream object
	streamBuilder, ok := builder.(Stream)

	if !ok {
		return Halt().WithError(derp.InternalError(location, "step: AddTags can only be used on a Stream"))
	}

	factory := streamBuilder.factory()
	streamService := factory.Stream()
	contentService := factory.Content()

	stream := streamBuilder._stream

	if step.Format != "" {
		stream.Content = contentService.New(step.Format, stream.Content.Raw)
	}

	if step.RemoveHTML {
		stream.Content.HTML = html.RemoveAnchors(stream.Content.HTML)
	}

	if step.AddLinks {
		contentService.ApplyLinks(&stream.Content)
	}

	if step.AddTags {
		streamService.CalculateTags(builder.session(), stream)

		if step.TagPath != "" {
			contentService.ApplyTags(&stream.Content, step.TagPath, stream.Hashtags)
		}
	}

	return Continue()
}
