package step

import (
	"github.com/benpate/rosetta/mapof"
)

// MarkNotificationsRead is a Step that marks all of the current User's notifications as read.
type MarkNotificationsRead struct{}

// NewMarkNotificationsRead returns a fully initialized MarkNotificationsRead object
func NewMarkNotificationsRead(stepInfo mapof.Any) (MarkNotificationsRead, error) {
	return MarkNotificationsRead{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step MarkNotificationsRead) Name() string {
	return "mark-notifications-read"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step MarkNotificationsRead) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step MarkNotificationsRead) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step MarkNotificationsRead) RequiredRoles() []string {
	return []string{}
}
