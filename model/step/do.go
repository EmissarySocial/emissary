package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Do is an action-step that calls anoter action to continue processing
type Do struct {
	Action string
}

// NewDo returns a fully initialized Do object
func NewDo(stepInfo mapof.Any) (Do, error) {
	return Do{
		Action: stepInfo.GetString("action"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Do) AmStep() {}
