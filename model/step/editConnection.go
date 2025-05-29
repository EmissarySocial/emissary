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

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditConnection) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditConnection) RequiredRoles() []string {
	return []string{}
}
