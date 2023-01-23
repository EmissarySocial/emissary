package step

import "github.com/benpate/rosetta/mapof"

// EditFeatures contains the configuration data for a modal that lets users edit the features attached to a stream.
type EditFeatures struct{}

func NewEditFeatures(stepInfo mapof.Any) (EditFeatures, error) {
	return EditFeatures{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step EditFeatures) AmStep() {}
