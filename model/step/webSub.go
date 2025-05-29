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

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WebSub) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WebSub) RequiredRoles() []string {
	return []string{}
}
