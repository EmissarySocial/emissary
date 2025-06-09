package step

import "github.com/benpate/rosetta/mapof"

// ReloadPage represents an pipeline-step that forwards the user to a new page.
type ReloadPage struct{}

// NewReloadPage returns a fully initialized ReloadPage object
func NewReloadPage(stepInfo mapof.Any) (ReloadPage, error) {
	return ReloadPage{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ReloadPage) Name() string {
	return "reload-page"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ReloadPage) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ReloadPage) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ReloadPage) RequiredRoles() []string {
	return []string{}
}
