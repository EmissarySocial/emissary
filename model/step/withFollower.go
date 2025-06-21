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

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithFollower) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithFollower) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithFollower) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
