package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SaveAndPublish is a Step that makes the signed-in User the author of a Stream, moves it
// into a state, and saves it.  The state's "outbox" flag decides whether followers are sent it.
type SaveAndPublish struct {
	StateID   string // The ID of the state that this step moves the Stream into.
	Republish bool   // If TRUE, sends this Stream to every syndication target again.
}

// NewSaveAndPublish returns a fully initialized SaveAndPublish object
func NewSaveAndPublish(stepInfo mapof.Any) (SaveAndPublish, error) {

	result := SaveAndPublish{
		StateID:   first(stepInfo.GetString("state"), "published"),
		Republish: stepInfo.GetBool("republish"),
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SaveAndPublish) Name() string {
	return "save-and-publish"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SaveAndPublish) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SaveAndPublish) RequiredStates() []string {
	return []string{step.StateID}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SaveAndPublish) RequiredRoles() []string {
	return []string{}
}
