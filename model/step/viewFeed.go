package step

import "github.com/benpate/rosetta/mapof"

// ViewFeed represents an action-step that can render a Stream into HTML
type ViewFeed struct {
}

// NewViewFeed generates a fully initialized ViewFeed step.
func NewViewFeed(stepInfo mapof.Any) (ViewFeed, error) {

	return ViewFeed{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step ViewFeed) AmStep() {}
