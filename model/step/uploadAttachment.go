package step

import "github.com/benpate/rosetta/mapof"

// UploadAttachment represents an action that can upload attachments.  It can only be used on a StreamBuilder
type UploadAttachment struct {
	Maximum int
}

// NewUploadAttachment returns a fully parsed UploadAttachment object
func NewUploadAttachment(stepInfo mapof.Any) (UploadAttachment, error) {
	return UploadAttachment{
		Maximum: stepInfo.GetInt("maximum"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step UploadAttachment) AmStep() {}
