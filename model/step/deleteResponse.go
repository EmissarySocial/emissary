package step

import "github.com/benpate/rosetta/mapof"

// DeleteResponse represents an action that can upload attachments.  It can only be used on a StreamRenderer
type DeleteResponse struct{}

// NewDeleteResponse returns a fully parsed DeleteResponse object
func NewDeleteResponse(stepInfo mapof.Any) (DeleteResponse, error) {
	return DeleteResponse{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step DeleteResponse) AmStep() {}
