package step

import (
	"github.com/benpate/rosetta/mapof"
)

// EditRegistration is a Step that locates an existing widget and
// creates a builder for it.
type EditRegistration struct{}

// NewEditRegistration returns a fully initialized EditRegistration object
func NewEditRegistration(stepInfo mapof.Any) (EditRegistration, error) {
	return EditRegistration{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step EditRegistration) Name() string {
	return "edit-registration"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step EditRegistration) RequiredModel() string {
	return "Domain"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditRegistration) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditRegistration) RequiredRoles() []string {
	return []string{}
}
