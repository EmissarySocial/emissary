package render

import (
	"io"

	"github.com/benpate/datatype"
)

// StepSetMentions represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetMentions struct {
	paths []string
}

func NewStepSetMentions(stepInfo datatype.Map) StepSetMentions {

	return StepSetMentions{
		paths: stepInfo.GetSliceOfString("paths"),
	}
}

// Get does not display anything.
func (step StepSetMentions) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post triggers a process to scan the designated paths for potential WebMentions, and adds them to the document if available.
func (step StepSetMentions) Post(buffer io.Writer, renderer Renderer) error {
	return nil
}
