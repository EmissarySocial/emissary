package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// ViewJSONLD represents an action-step that can build a Stream into HTML
type ViewJSONLD struct {
	Method string
}

// NewViewJSONLD generates a fully initialized ViewJSONLD step.
func NewViewJSONLD(stepInfo mapof.Any) (ViewJSONLD, error) {

	return ViewJSONLD{
		Method: first.String(stepInfo.GetString("method"), "get"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step ViewJSONLD) AmStep() {}
