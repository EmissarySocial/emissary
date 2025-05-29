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

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ReloadPage) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ReloadPage) RequiredRoles() []string {
	return []string{}
}
