package step

import "github.com/benpate/rosetta/mapof"

// SetState is a Step that can change a Stream's state
type SetState struct {
	State string
}

func NewSetState(stepInfo mapof.Any) (SetState, error) {

	return SetState{
		State: stepInfo.GetString("state"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetState) AmStep() {}
