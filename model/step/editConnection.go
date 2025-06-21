package step

import "github.com/benpate/rosetta/mapof"

// EditConnection contains the configuration data for a modal that lets administrators manage connections to external servers.
type EditConnection struct{}

func NewEditConnection(stepInfo mapof.Any) (EditConnection, error) {
	return EditConnection{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step EditConnection) Name() string {
	return "edit-connection"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step EditConnection) RequiredModel() string {
	return "Domain"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditConnection) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditConnection) RequiredRoles() []string {
	return []string{}
}
