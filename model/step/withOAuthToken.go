package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithOAuthToken is a Step executes a list of sub-steps on every child of the current Stream
type WithOAuthToken struct {
	SubSteps []Step
}

// NewWithOAuthToken returns a fully initialized WithOAuthToken object
func NewWithOAuthToken(stepInfo mapof.Any) (WithOAuthToken, error) {

	const location = "NewWithOAuthToken"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithOAuthToken{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithOAuthToken{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithOAuthToken) Name() string {
	return "with-oauth-token"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithOAuthToken) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithOAuthToken) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithOAuthToken) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
