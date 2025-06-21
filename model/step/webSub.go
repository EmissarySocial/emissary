package step

import "github.com/benpate/rosetta/mapof"

// WebSub is a Step that can build a Stream into HTML
type WebSub struct {
}

// NewWebSub generates a fully initialized WebSub step.
func NewWebSub(stepInfo mapof.Any) (WebSub, error) {
	return WebSub{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WebSub) Name() string {
	return "web-sub"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WebSub) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WebSub) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WebSub) RequiredRoles() []string {
	return []string{}
}
