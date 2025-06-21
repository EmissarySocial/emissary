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

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step RefreshPage) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step RefreshPage) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step RefreshPage) RequiredRoles() []string {
	return []string{}
}
