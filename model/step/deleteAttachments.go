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

// AmStep is here only to verify that this struct is a build pipeline step
func (step DeleteAttachments) AmStep() {}
