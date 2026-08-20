package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SortWidgets is a Step that can update multiple records at once
type SortWidgets struct{}

// NewSortWidgets returns a fully initialized SortWidgets step, or an error if its configuration is invalid
func NewSortWidgets(stepInfo mapof.Any) (SortWidgets, error) {

	return SortWidgets{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SortWidgets) Name() string {
	return "sort-widgets"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SortWidgets) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SortWidgets) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SortWidgets) RequiredRoles() []string {
	return []string{}
}
