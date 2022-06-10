package step

import (
	"github.com/benpate/datatype"
)

// DeleteAttachment represents an action that can upload attachments.  It can only be used on a StreamRenderer
type DeleteAttachment struct{}

// NewDeleteAttachment returns a fully parsed DeleteAttachment object
func NewDeleteAttachment(stepInfo datatype.Map) (DeleteAttachment, error) {
	return DeleteAttachment{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step DeleteAttachment) AmStep() {}
