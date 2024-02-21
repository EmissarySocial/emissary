package step

import (
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// ProcessContent is an action that can add new sub-streams to the domain.
type ProcessContent struct {
	RemoveHTML bool
	AddTags    bool
	AddLinks   bool
}

// NewProcessContent returns a fully initialized ProcessContent record
func NewProcessContent(stepInfo mapof.Any) (ProcessContent, error) {
	return ProcessContent{
		RemoveHTML: convert.BoolDefault(stepInfo["remove-html"], true),
		AddTags:    convert.BoolDefault(stepInfo["add-tags"], true),
		AddLinks:   convert.BoolDefault(stepInfo["add-links"], true),
	}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step ProcessContent) AmStep() {}
