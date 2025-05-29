package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithRule is a Step that returns a new Rule Builder
type WithRule struct {
	SubSteps []Step
}

// NewWithRule returns a fully initialized WithRule object
func NewWithRule(stepInfo mapof.Any) (WithRule, error) {

	const location = "NewWithRule"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithRule{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithRule{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithRule) Name() string {
	return "with-rule"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithRule) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithRule) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
