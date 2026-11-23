package step

import (
	"github.com/benpate/rosetta/mapof"
)

// StartupCreateStreams is a Step that seeds an empty Domain with the Streams that its Theme
// defines in "startupStreams", and sets the landing pages in "startupDefaultPages".
type StartupCreateStreams struct{}

// NewStartupCreateStreams returns a fully initialized StartupCreateStreams object
func NewStartupCreateStreams(stepInfo mapof.Any) (StartupCreateStreams, error) {
	return StartupCreateStreams{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step StartupCreateStreams) Name() string {
	return "startup-create-streams"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step StartupCreateStreams) RequiredModel() string {
	return "Domain"
}

// RequiredTemplateRoles returns the template roles that a Template MUST declare in order to
// use this Step.  Seeding a Domain with content is an administrative act, so this Step is
// restricted to admin Templates -- "Domain" alone would still admit a public-facing Template.
func (step StartupCreateStreams) RequiredTemplateRoles() []string {
	return []string{"admin"}
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step StartupCreateStreams) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step StartupCreateStreams) RequiredRoles() []string {
	return []string{}
}
