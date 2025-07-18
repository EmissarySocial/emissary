package step

import (
	"github.com/benpate/rosetta/mapof"
)

// EditWidget is a Step that locates an existing widget and
// creates a builder for it.
type EditWidget struct{}

// NewEditWidget returns a fully initialized EditWidget object
func NewEditWidget(stepInfo mapof.Any) (EditWidget, error) {
	return EditWidget{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step EditWidget) Name() string {
	return "edit-widget"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step EditWidget) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditWidget) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditWidget) RequiredRoles() []string {
	return []string{}
}
