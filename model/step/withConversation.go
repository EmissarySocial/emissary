package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithConversation is a Step executes a list of sub-steps on every child of the current Stream
type WithConversation struct {
	SubSteps []Step
}

// NewWithConversation returns a fully initialized WithConversation object
func NewWithConversation(stepInfo mapof.Any) (WithConversation, error) {

	const location = "NewWithConversation"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithConversation{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithConversation{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithConversation) Name() string {
	return "with-conversation"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithConversation) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithConversation) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithConversation) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
