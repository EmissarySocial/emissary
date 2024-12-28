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

func (step StepSleep) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	time.Sleep(time.Duration(step.Duration) * time.Millisecond)
	return nil
}

func (step StepSleep) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	time.Sleep(time.Duration(step.Duration) * time.Millisecond)
	return nil
}
