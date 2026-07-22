package build

import (
	"io"
)

// StepProcessTags is a DEPRECATED action step. #hashtags are now extracted automatically when a
// Stream or User is saved, so this step does nothing. It is retained only so that older Templates
// that still reference "process-tags" continue to load (see model.step.NewProcessTags, which logs
// the deprecation warning at Template-load time).
type StepProcessTags struct {
	Paths []string
}

// Get is a no-op for this deprecated step.
func (step StepProcessTags) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post is a no-op for this deprecated step. Tag processing now happens automatically on save.
func (step StepProcessTags) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return Continue()
}
