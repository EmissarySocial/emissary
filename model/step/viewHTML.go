package step

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// ViewHTML represents an action-step that can render a Stream into HTML
type ViewHTML struct {
	File   string
	Method sliceof.String
}

// NewViewHTML generates a fully initialized ViewHTML step.
func NewViewHTML(stepInfo mapof.Any) (ViewHTML, error) {

	method := stepInfo.GetSliceOfString("method")

	if len(method) == 0 {
		method = []string{"get"}
	}

	return ViewHTML{
		File:   stepInfo.GetString("file"),
		Method: method,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step ViewHTML) AmStep() {}
