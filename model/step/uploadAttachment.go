package step

import "github.com/benpate/rosetta/maps"

// UploadAttachment represents an action that can upload attachments.  It can only be used on a StreamRenderer
type UploadAttachment struct {
	Maximum int
}

// NewUploadAttachment returns a fully parsed UploadAttachment object
func NewUploadAttachment(stepInfo maps.Map) (UploadAttachment, error) {
	return UploadAttachment{
		Maximum: stepInfo.GetInt("maximum"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step UploadAttachment) AmStep() {}
