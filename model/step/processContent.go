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

// Name returns the name of the step, which is used in debugging.
func (step ProcessContent) Name() string {
	return "process-content"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ProcessContent) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ProcessContent) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ProcessContent) RequiredRoles() []string {
	return []string{}
}
