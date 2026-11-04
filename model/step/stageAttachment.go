package step

import (
	"github.com/benpate/rosetta/mapof"
)

// StageAttachment is a Step that holds one file under the signed-in User for an object that does
// not exist yet, and answers with the file's tile for the upload-multiple widget
type StageAttachment struct{}

// NewStageAttachment returns a fully parsed StageAttachment object
func NewStageAttachment(_ mapof.Any) (StageAttachment, error) {
	return StageAttachment{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step StageAttachment) Name() string {
	return "stage-attachment"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step StageAttachment) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step StageAttachment) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step StageAttachment) RequiredRoles() []string {
	return []string{}
}
