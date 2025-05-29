package step

import (
	"github.com/benpate/rosetta/mapof"
)

// ViewHTML is a Step that can build a Stream into HTML
type ViewHTML struct {
	File       string
	Method     string
	AsFullPage bool
}

// NewViewHTML generates a fully initialized ViewHTML step.
func NewViewHTML(stepInfo mapof.Any) (ViewHTML, error) {

	return ViewHTML{
		File:       stepInfo.GetString("file"),
		Method:     first(stepInfo.GetString("method"), "get"),
		AsFullPage: stepInfo.GetBool("as-full-page"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ViewHTML) Name() string {
	return "view-html"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ViewHTML) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ViewHTML) RequiredRoles() []string {
	return []string{}
}
