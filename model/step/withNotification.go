package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithNotification is a Step that returns a new Notification Builder
type WithNotification struct {
	SubSteps []Step
}

// NewWithNotification returns a fully initialized WithNotification object
func NewWithNotification(stepInfo mapof.Any) (WithNotification, error) {

	const location = "NewWithNotification"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithNotification{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithNotification{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithNotification) Name() string {
	return "with-notification"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithNotification) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithNotification) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithNotification) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
