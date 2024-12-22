package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SortWidgets is an action-step that can update multiple records at once
type SortWidgets struct{}

func NewSortWidgets(stepInfo mapof.Any) (SortWidgets, error) {

	return SortWidgets{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SortWidgets) AmStep() {}
