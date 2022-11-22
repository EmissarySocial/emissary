package step

import "github.com/benpate/rosetta/maps"

// SetThumbnail represents an action-step that can update the data.DataMap custom data stored in a Stream
type SetThumbnail struct {
	Path string
}

func NewSetThumbnail(stepInfo maps.Map) (SetThumbnail, error) {
	return SetThumbnail{
		Path: stepInfo.GetString("path"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetThumbnail) AmStep() {}
