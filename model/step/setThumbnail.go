package step

import (
	"github.com/benpate/datatype"
)

// SetThumbnail represents an action-step that can update the data.DataMap custom data stored in a Stream
type SetThumbnail struct{}

func NewSetThumbnail(stepInfo datatype.Map) (SetThumbnail, error) {
	return SetThumbnail{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetThumbnail) AmStep() {}
