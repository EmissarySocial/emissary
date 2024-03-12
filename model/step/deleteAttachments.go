package step

import "github.com/benpate/rosetta/mapof"

// DeleteAttachments represents an action that can upload attachments.  It can only be used on a StreamBuilder
type DeleteAttachments struct {
	All bool
}

// NewDeleteAttachments returns a fully parsed DeleteAttachments object
func NewDeleteAttachments(stepInfo mapof.Any) (DeleteAttachments, error) {
	return DeleteAttachments{
		All: stepInfo.GetBool("all"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step DeleteAttachments) AmStep() {}
