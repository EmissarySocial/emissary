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

// AmStep is here only to verify that this struct is a build pipeline step
func (step EditRegistration) AmStep() {}

func (step EditRegistration) RequireType() string {
	return "registration"
}
