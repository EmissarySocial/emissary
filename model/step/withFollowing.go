package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithFollowing is a Step that returns a new Following builder
type WithFollowing struct {
	SubSteps []Step
}

// NewWithFollowing returns a fully initialized WithFollowing object
func NewWithFollowing(stepInfo mapof.Any) (WithFollowing, error) {

	const location = "NewWithFollowing"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithFollowing{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithFollowing{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithFollowing) Name() string {
	return "with-following"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithFollowing) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithFollowing) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithFollowing) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
