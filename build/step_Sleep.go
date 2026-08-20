package build

import (
	"io"
	"time"
)

// StepSleep is a Step that sleeps for a determined period of time.
// It should really only be used for debugging.
type StepSleep struct {
	Duration int
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSleep) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	time.Sleep(time.Duration(step.Duration) * time.Millisecond)
	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepSleep) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	time.Sleep(time.Duration(step.Duration) * time.Millisecond)
	return nil
}
