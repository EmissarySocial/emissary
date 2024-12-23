package build

import (
	"io"
	"time"

	"github.com/davecgh/go-spew/spew"
)

// StepSleep is a Step that sleeps for a determined period of time.
// It should really only be used for debugging.
type StepSleep struct {
	Duration int
}

func (step StepSleep) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	spew.Dump("Sleeping", step.Duration)
	time.Sleep(time.Duration(step.Duration) * time.Millisecond)
	return nil
}

func (step StepSleep) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	spew.Dump("Sleeping", step.Duration)
	time.Sleep(time.Duration(step.Duration) * time.Millisecond)
	return nil
}
