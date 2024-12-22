package step

import (
	"github.com/benpate/rosetta/mapof"
)

// EditWidget is an action-step that locates an existing widget and
// creates a builder for it.
type EditWidget struct{}

// NewEditWidget returns a fully initialized EditWidget object
func NewEditWidget(stepInfo mapof.Any) (EditWidget, error) {
	return EditWidget{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step EditWidget) AmStep() {}
