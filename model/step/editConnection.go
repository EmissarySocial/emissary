package step

import "github.com/benpate/rosetta/maps"

// EditConnection contains the configuration data for a modal that lets administrators manage connections to external servers.
type EditConnection struct{}

func NewEditConnection(stepInfo maps.Map) (EditConnection, error) {
	return EditConnection{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step EditConnection) AmStep() {}
