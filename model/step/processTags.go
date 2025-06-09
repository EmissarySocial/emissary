package step

import (
	"strings"

	"github.com/benpate/rosetta/mapof"
)

// ProcessTags is an action that can add new sub-streams to the domain.
type ProcessTags struct {
	Paths []string
}

// NewProcessTags returns a fully initialized ProcessTags record
func NewProcessTags(stepInfo mapof.Any) (ProcessTags, error) {

	pathString := stepInfo.GetString("paths")
	pathSlice := strings.Split(pathString, ",")

	for index := range pathSlice {
		pathSlice[index] = strings.TrimSpace(pathSlice[index])
	}

	return ProcessTags{
		Paths: pathSlice,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ProcessTags) Name() string {
	return "process-tags"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ProcessTags) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ProcessTags) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ProcessTags) RequiredRoles() []string {
	return []string{}
}
