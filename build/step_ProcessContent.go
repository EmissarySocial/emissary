package build

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/html"
)

// StepProcessContent is an action step that reformats a Stream's content (format conversion, HTML
// removal, link detection).  The AddTags and TagPath fields are DEPRECATED and ignored: #hashtags
// are now extracted and linkified automatically in Stream.Save.  The fields remain so this struct
// stays convertible from model.step.ProcessContent.
type StepProcessContent struct {
	Format     string
	RemoveHTML bool
	AddLinks   bool
	AddTags    bool   // Deprecated: #hashtags are processed automatically in Stream.Save
	TagPath    string // Deprecated: #hashtags are processed automatically in Stream.Save
}

// Get builds the HTML for this step - either a modal template selector, or the embedded edit form
func (step StepProcessContent) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepProcessContent) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepProcessContent.Post"

	// Require that we are working with a Stream object
	streamBuilder, ok := builder.(Stream)

	if !ok {
		return Halt().WithError(derp.Internal(location, "step: AddTags can only be used on a Stream"))
	}

	factory := streamBuilder.factory()
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

	// NOTE: AddTags/TagPath are intentionally ignored here. #hashtags are extracted and linkified
	// automatically in Stream.Save (driven by the Template's tagPaths / tagUrl).

	return Continue()
}
