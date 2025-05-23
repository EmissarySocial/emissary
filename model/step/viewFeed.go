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

// AmStep is here only to verify that this struct is a build pipeline step
func (step ViewFeed) AmStep() {}
