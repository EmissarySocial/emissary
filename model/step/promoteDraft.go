package step

import (
	"github.com/benpate/rosetta/mapof"
)

// StreamPromoteDraft represents a pipeline-step that can copy the Container from a StreamDraft into its corresponding Stream
type StreamPromoteDraft struct {
	StateID string
}

func NewStreamPromoteDraft(stepInfo mapof.Any) (StreamPromoteDraft, error) {
	return StreamPromoteDraft{
		StateID: first(stepInfo.GetString("state"), "published"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step StreamPromoteDraft) Name() string {
	return "promote-draft"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step StreamPromoteDraft) RequiredStates() []string {
	return []string{step.StateID}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step StreamPromoteDraft) RequiredRoles() []string {
	return []string{}
}
