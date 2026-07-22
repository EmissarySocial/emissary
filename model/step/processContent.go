package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
)

// ProcessContent is an action that reformats a Stream's content (format conversion, HTML removal,
// link detection).  The AddTags and TagPath fields are DEPRECATED and ignored: #hashtags are now
// extracted and linkified automatically when a Stream is saved.  They are still parsed so that
// older Templates that set them continue to load.
type ProcessContent struct {
	Format     string
	RemoveHTML bool
	AddLinks   bool
	AddTags    bool   // Deprecated: #hashtags are processed automatically in Stream.Save
	TagPath    string // Deprecated: #hashtags are processed automatically in Stream.Save
}

// NewProcessContent returns a fully initialized ProcessContent record
func NewProcessContent(stepInfo mapof.Any) (ProcessContent, error) {

	format := stepInfo.GetString("format")

	if allowed := (sliceof.String{"", "MARKDOWN", "EDITORJS", "HTML"}); allowed.NotContains(format) {
		return ProcessContent{}, derp.Validation("Format must be one of [MARKDOWN, EDITORJS, HTML]")
	}

	addTags := stepInfo.GetBool("add-tags")
	tagPath := stepInfo.GetString("tag-path")

	// RULE: "add-tags"/"tag-path" are deprecated. Warn once at Template-load time so operators know to remove them.
	if addTags || tagPath != "" {
		log.Warn().Str("location", "model.step.NewProcessContent").Msg("the \"add-tags\"/\"tag-path\" options on \"process-content\" are deprecated and ignored; remove them — #hashtags are processed automatically on save")
	}

	return ProcessContent{
		Format:     format,
		RemoveHTML: stepInfo.GetBool("remove-html"),
		AddLinks:   stepInfo.GetBool("add-links"),
		AddTags:    addTags,
		TagPath:    tagPath,
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
