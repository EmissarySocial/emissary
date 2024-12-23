package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Sleep is a Step that sleeps for a determined amount of time.
// It should really only be used for debugging.
type Sleep struct {
	Duration int
}

// NewSleep returns a fully initialized Sleep object
func NewSleep(stepInfo mapof.Any) (Sleep, error) {

	return Sleep{
		Duration: stepInfo.GetInt("duration"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Sleep) AmStep() {}
