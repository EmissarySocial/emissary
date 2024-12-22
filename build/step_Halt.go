package build

import (
	"io"
)

// StepHalt is an action-step that can save changes to any object
type StepHalt struct{}

func (step StepHalt) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return Halt()
}

// Post saves the object to the database
func (step StepHalt) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Halt()
}
