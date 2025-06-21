package step

import (
	"github.com/benpate/rosetta/mapof"
)

// ViewFeed is a Step that can build a Stream into HTML
type ViewFeed struct {
	SearchTypes []string
}

// NewViewFeed generates a fully initialized ViewFeed step.
func NewViewFeed(stepInfo mapof.Any) (ViewFeed, error) {

	return ViewFeed{
		SearchTypes: stepInfo.GetSliceOfString("search-types"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ViewFeed) Name() string {
	return "view-feed"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ViewFeed) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ViewFeed) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ViewFeed) RequiredRoles() []string {
	return []string{}
}
