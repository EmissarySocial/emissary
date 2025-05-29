package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SaveAndPublish is a Step that can update a stream's SaveAndPublishDate with the current time.
type SaveAndPublish struct {
	StateID   string // The ID of the state that this step will update.
	Outbox    bool   // If TRUE, also send updates to this User's outbox.
	Republish bool   // If TRUE, republishes this stream to syndication targets.
}

// NewSaveAndPublish returns a fully initialized SaveAndPublish object
func NewSaveAndPublish(stepInfo mapof.Any) (SaveAndPublish, error) {

	result := SaveAndPublish{
		StateID:   first(stepInfo.GetString("state"), "published"),
		Outbox:    stepInfo.GetBool("outbox"),
		Republish: stepInfo.GetBool("republish"),
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SaveAndPublish) Name() string {
	return "save-and-publish"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SaveAndPublish) RequiredStates() []string {
	return []string{step.StateID}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SaveAndPublish) RequiredRoles() []string {
	return []string{}
}
