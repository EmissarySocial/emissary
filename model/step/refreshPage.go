package step

import "github.com/benpate/rosetta/mapof"

// RefreshPage represents an pipeline-step that forwards the user to a new page.
type RefreshPage struct{}

// NewRefreshPage returns a fully initialized RefreshPage object
func NewRefreshPage(stepInfo mapof.Any) (RefreshPage, error) {
	return RefreshPage{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step RefreshPage) Name() string {
	return "refresh-page"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step RefreshPage) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step RefreshPage) RequiredRoles() []string {
	return []string{}
}
