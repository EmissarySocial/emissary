package step

import (
	"github.com/benpate/datatype"
)

// UploadAttachment represents an action that can upload attachments.  It can only be used on a StreamRenderer
type UploadAttachment struct{}

// NewUploadAttachment returns a fully parsed UploadAttachment object
func NewUploadAttachment(stepInfo datatype.Map) (UploadAttachment, error) {
	return UploadAttachment{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step UploadAttachment) AmStep() {}
