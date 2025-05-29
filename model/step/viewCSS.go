package step

import (
	"github.com/benpate/rosetta/mapof"
)

// ViewCSS is a Step that can render a Stream into a CSS stylesheet
type ViewCSS struct {
	File string
}

// NewViewCSS generates a fully initialized ViewCSS step.
func NewViewCSS(stepInfo mapof.Any) (ViewCSS, error) {

	return ViewCSS{
		File: stepInfo.GetString("file"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ViewCSS) Name() string {
	return "view-css"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ViewCSS) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ViewCSS) RequiredRoles() []string {
	return []string{}
}
