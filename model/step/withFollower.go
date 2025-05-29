package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithFollower is a Step that returns a new Follower Builder
type WithFollower struct {
	SubSteps []Step
}

// NewWithFollower returns a fully initialized WithFollower object
func NewWithFollower(stepInfo mapof.Any) (WithFollower, error) {

	const location = "NewWithFollower"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithFollower{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithFollower{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithFollower) Name() string {
	return "with-follower"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithFollower) RequiredStates() []string {
	return requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithFollower) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
