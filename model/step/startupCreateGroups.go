package step

import (
	"github.com/benpate/rosetta/mapof"
)

// StartupCreateGroups is a Step that seeds a Domain that has no Groups with the Groups that
// its Theme defines in "startupGroups".
type StartupCreateGroups struct{}

// NewStartupCreateGroups returns a fully initialized StartupCreateGroups object
func NewStartupCreateGroups(stepInfo mapof.Any) (StartupCreateGroups, error) {
	return StartupCreateGroups{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step StartupCreateGroups) Name() string {
	return "startup-create-groups"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step StartupCreateGroups) RequiredModel() string {
	return "Domain"
}

// RequiredTemplateRoles returns the template roles that a Template MUST declare in order to
// use this Step.  Seeding a Domain with Groups is an administrative act.
func (step StartupCreateGroups) RequiredTemplateRoles() []string {
	return []string{"admin"}
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step StartupCreateGroups) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step StartupCreateGroups) RequiredRoles() []string {
	return []string{}
}
