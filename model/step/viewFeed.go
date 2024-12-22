package step

import "github.com/benpate/rosetta/mapof"

// ViewFeed is an action-step that can build a Stream into HTML
type ViewFeed struct {
}

// NewViewFeed generates a fully initialized ViewFeed step.
func NewViewFeed(stepInfo mapof.Any) (ViewFeed, error) {

	return ViewFeed{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step ViewFeed) AmStep() {}
