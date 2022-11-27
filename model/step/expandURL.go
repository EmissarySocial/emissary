package step

import (
	"github.com/benpate/rosetta/maps"
)

// ExpandURL is an action that can add new sub-streams to the domain.
type ExpandURL struct {
	Path string
}

// NewExpandURL returns a fully initialized ExpandURL record
func NewExpandURL(stepInfo maps.Map) (ExpandURL, error) {

	return ExpandURL{
		Path: stepInfo.GetString("path"),
	}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step ExpandURL) AmStep() {}
