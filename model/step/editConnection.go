package step

import "github.com/benpate/rosetta/mapof"

// EditConnection contains the configuration data for a modal that lets administrators manage connections to external servers.
type EditConnection struct{}

func NewEditConnection(stepInfo mapof.Any) (EditConnection, error) {
	return EditConnection{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step EditConnection) AmStep() {}
