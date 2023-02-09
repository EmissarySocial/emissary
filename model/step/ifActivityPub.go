package step

import (
	"github.com/benpate/rosetta/mapof"
)

// IfActivityPub represents an action-step that can update the data.DataMap custom data stored in a Stream
type IfActivityPub struct {
	View    string
	Dataset string
}

func NewIfActivityPub(stepInfo mapof.Any) (IfActivityPub, error) {
	return IfActivityPub{
		View:    stepInfo.GetString("view"),
		Dataset: stepInfo.GetString("dataset"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step IfActivityPub) AmStep() {}
