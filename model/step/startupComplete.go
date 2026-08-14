package step

import (
	"github.com/benpate/rosetta/mapof"
)

// StartupComplete is a Step that ends the startup wizard, moving the Domain out of its
// "STARTUP" state and into production.
type StartupComplete struct{}

// NewStartupComplete returns a fully initialized StartupComplete object
func NewStartupComplete(stepInfo mapof.Any) (StartupComplete, error) {
	return StartupComplete{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step StartupComplete) Name() string {
	return "startup-complete"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step StartupComplete) RequiredModel() string {
	return "Domain"
}

// RequiredTemplateRoles returns the template roles that a Template MUST declare in order to
// use this Step.  Ending setup opens the whole Domain to the public, so this Step is restricted
// to admin Templates -- "Domain" alone would still admit a public-facing Template.
func (step StartupComplete) RequiredTemplateRoles() []string {
	return []string{"admin"}
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step StartupComplete) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step StartupComplete) RequiredRoles() []string {
	return []string{}
}
