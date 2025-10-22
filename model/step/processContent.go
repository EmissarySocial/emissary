package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// ProcessContent is an action that can add new sub-streams to the domain.
type ProcessContent struct {
	Format     string
	RemoveHTML bool
	AddLinks   bool
	AddTags    bool
	TagPath    string
}

// NewProcessContent returns a fully initialized ProcessContent record
func NewProcessContent(stepInfo mapof.Any) (ProcessContent, error) {

	format := stepInfo.GetString("format")
	allowed := sliceof.String{"", "MARKDOWN", "EDITORJS", "HTML"}

	if !allowed.Contains(format) {
		return ProcessContent{}, derp.ValidationError("Format must be one of [MARKDOWN, EDITORJS, HTML]")
	}

	return ProcessContent{
		Format:     format,
		RemoveHTML: stepInfo.GetBool("remove-html"),
		AddLinks:   stepInfo.GetBool("add-links"),
		AddTags:    stepInfo.GetBool("add-tags"),
		TagPath:    stepInfo.GetString("tag-path"),
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
