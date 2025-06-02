package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithPrivilege is a Step executes a list of sub-steps on every child of the current Stream
type WithPrivilege struct {
	SubSteps []Step
}

// NewWithPrivilege returns a fully initialized WithPrivilege object
func NewWithPrivilege(stepInfo mapof.Any) (WithPrivilege, error) {

	const location = "NewWithPrivilege"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithPrivilege{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithPrivilege{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithPrivilege) Name() string {
	return "with-privilege"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithPrivilege) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithPrivilege) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
