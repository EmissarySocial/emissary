package step

import "github.com/benpate/rosetta/maps"

// DeleteAttachments represents an action that can upload attachments.  It can only be used on a StreamRenderer
type DeleteAttachments struct {
	All bool
}

// NewDeleteAttachments returns a fully parsed DeleteAttachments object
func NewDeleteAttachments(stepInfo maps.Map) (DeleteAttachments, error) {
	return DeleteAttachments{
		All: stepInfo.GetBool("all"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step DeleteAttachments) AmStep() {}
