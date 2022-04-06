package step

import (
	"github.com/benpate/datatype"
)

// ViewHTML represents an action-step that can render a Stream into HTML
type ViewHTML struct {
	Filename string
}

// NewViewHTML generates a fully initialized ViewHTML step.
func NewViewHTML(stepInfo datatype.Map) (ViewHTML, error) {
	return ViewHTML{
		Filename: stepInfo.GetString("file"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step ViewHTML) AmStep() {}
