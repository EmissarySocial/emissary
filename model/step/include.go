package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Include is a Step that runs another action's steps on the current object
type Include struct {
	Action string
	Method string
}

// NewInclude returns a fully initialized Include object
func NewInclude(stepInfo mapof.Any) (Include, error) {

	// An include draws on a GET as often as it saves on a POST, so it runs on both by default
	method, err := parseMethod(stepInfo, "both")

	if err != nil {
		return Include{}, derp.Wrap(err, "model.step.NewInclude", "Invalid 'method'", stepInfo.GetString("action"))
	}

	return Include{
		Action: stepInfo.GetString("action"),
		Method: method,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Include) Name() string {
	return "include"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step Include) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Include) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Include) RequiredRoles() []string {
	return []string{}
}
