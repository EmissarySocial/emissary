package render

import (
	"io"
)

// StepHalt represents an action-step that can save changes to any object
type StepHalt struct{}

func (step StepHalt) Get(renderer Renderer, _ io.Writer) ExitCondition {
	return ExitHalt()
}

// Post saves the object to the database
func (step StepHalt) Post(renderer Renderer, _ io.Writer) ExitCondition {
	return ExitHalt()
}
