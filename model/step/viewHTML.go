package step

import "github.com/benpate/rosetta/maps"

// ViewHTML represents an action-step that can render a Stream into HTML
type ViewHTML struct {
	File string
}

// NewViewHTML generates a fully initialized ViewHTML step.
func NewViewHTML(stepInfo maps.Map) (ViewHTML, error) {
	return ViewHTML{
		File: getValue(stepInfo.GetString("file")),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step ViewHTML) AmStep() {}
