package step

import "github.com/benpate/rosetta/mapof"

// DeleteAttachments represents an action that removes one or more attachments from an object.  The filter criteria
// in this step is further narrowed by the "attachmentId" query parameter, if present.
type DeleteAttachments struct {
	All      bool   // If TRUE, then ALL ATTACHMENTS for this object will be deleted.
	Field    string // If set, the the attachment named by this property will be deleted. (If zero, then NOOP)
	Category string // If set, then all attachments from a specified group are deleted.
}

// NewDeleteAttachments returns a fully parsed DeleteAttachments object
func NewDeleteAttachments(stepInfo mapof.Any) (DeleteAttachments, error) {
	return DeleteAttachments{
		All:      stepInfo.GetBool("all"),
		Field:    stepInfo.GetString("field"),
		Category: stepInfo.GetString("category"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step DeleteAttachments) Name() string {
	return "delete-attachments"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step DeleteAttachments) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step DeleteAttachments) RequiredRoles() []string {
	return []string{}
}
