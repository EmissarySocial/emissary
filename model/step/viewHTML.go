package step

import (
	"github.com/benpate/rosetta/mapof"
)

// ViewHTML represents an action-step that can build a Stream into HTML
type ViewHTML struct {
	File   string
	Method string
}

// NewViewHTML generates a fully initialized ViewHTML step.
func NewViewHTML(stepInfo mapof.Any) (ViewHTML, error) {

	return ViewHTML{
		File:   stepInfo.GetString("file"),
		Method: first(stepInfo.GetString("method"), "get"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step ViewHTML) AmStep() {}
