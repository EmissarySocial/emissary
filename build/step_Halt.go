package build

import (
	"io"
)

// StepHalt is a Step that can save changes to any object
type StepHalt struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepHalt) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return Halt()
}

// Post saves the object to the database
func (step StepHalt) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Halt()
}
