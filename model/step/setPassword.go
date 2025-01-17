package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetPassword is a Step that can update the custom data stored in a Stream
type SetPassword struct{}

// NewSetPassword returns a fully initialized SetPassword object
func NewSetPassword(stepInfo mapof.Any) (SetPassword, error) {

	return SetPassword{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetPassword) AmStep() {}
