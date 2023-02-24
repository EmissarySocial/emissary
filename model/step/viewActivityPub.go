package step

import (
	"github.com/benpate/rosetta/mapof"
)

// ViewActivityPub represents an action-step that can update the data.DataMap custom data stored in a Stream
type ViewActivityPub struct {
	File string
}

// NewViewActivityPub returns a fully initialized ViewActivityPub object
func NewViewActivityPub(stepInfo mapof.Any) (ViewActivityPub, error) {

	return ViewActivityPub{
		File: stepInfo.GetString("file"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step ViewActivityPub) AmStep() {}
