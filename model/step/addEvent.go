package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// AddEvent is a Step that removes an HX-Trigger event from the HTTP result
type AddEvent struct {
	Method string
	Event  string
}

// NewAddEvent returns a fully initialized AddEvent object
func NewAddEvent(stepInfo mapof.Any) (AddEvent, error) {

	method, err := parseMethod(stepInfo, "post")

	if err != nil {
		return AddEvent{}, derp.Wrap(err, "model.step.NewAddEvent", "Invalid 'method'", stepInfo)
	}

	return AddEvent{
		Method: method,
		Event:  stepInfo.GetString("event"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step AddEvent) Name() string {
	return "add-event"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step AddEvent) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step AddEvent) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step AddEvent) RequiredRoles() []string {
	return []string{}
}
