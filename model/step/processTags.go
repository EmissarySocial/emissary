package step

import (
	"strings"

	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
)

// ProcessTags is a DEPRECATED action step. #hashtags are now extracted automatically when a
// Stream or User is saved (driven by the Template's tagPaths), so this step no longer does
// anything. It is retained only so that older Templates that still reference it continue to load.
type ProcessTags struct {
	Paths []string
}

// NewProcessTags returns a fully initialized ProcessTags record
func NewProcessTags(stepInfo mapof.Any) (ProcessTags, error) {

	// RULE: This step is deprecated. Warn once at Template-load time so operators know to remove it.
	log.Warn().Str("location", "model.step.NewProcessTags").Msg("the \"process-tags\" step is deprecated and does nothing; remove it from your Template — #hashtags are extracted automatically on save")

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
