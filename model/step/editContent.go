package step

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
)

// The `max-length` step setting counts CHARACTERS (runes), which is what every other
// length limit in this codebase counts -- schema `maxLength`, model.ContentMaxLength, and
// the rune count the step itself enforces.  It was briefly denominated in kilobytes, which
// read well in a template but meant the same word measured two different things depending
// on which file you were in.  A template now writes the same number the schema does.
const (
	// editContentDefaultMaxLength is the per-step content limit (in runes) applied when a
	// template does not specify its own `max-length`.  It is generous enough for long-form
	// articles while still rejecting the multi-megabyte bodies that a storage-exhaustion
	// attack relies on.
	editContentDefaultMaxLength = 64 << 10 // 65,536 runes

	// editContentMaxLengthCeiling is the largest `max-length` (in runes) a template may
	// configure.  A template cannot allow more content than the system will actually store,
	// so this is kept in sync with model.ContentMaxLength (the schema-level cap on
	// `content.raw`/`content.html`).  It is duplicated here rather than imported because
	// model imports model/step, so model/step cannot import model without creating a cycle.
	editContentMaxLengthCeiling = 1 << 20 // 1,048,576 runes == model.ContentMaxLength (1 MiB)
)

// EditContent is a Step that can edit/update Container in a streamDraft.
type EditContent struct {
	Filename       string
	Fieldname      string
	Format         string
	MaxLength      int  // Maximum length (in runes) of the submitted content.  Submissions longer than this are rejected.
	RequireContent bool // If TRUE, then the submitted content must not be empty. Otherwise, the step halts with an error.
}

// NewEditContent parses and validates the configuration for an "edit-content" step,
// applying the default content limit and clamping it to the storage ceiling.  The
// `max-length` setting counts characters, and is used verbatim.
func NewEditContent(stepInfo mapof.Any) (EditContent, error) {

	const location = "model.step.NewEditContent"

	// Validate the step configuration
	if _, err := schema.New(StepEditContentSchema()).Validate(stepInfo); err != nil {
		return EditContent{}, err
	}

	// RULE: Apply the default limit when a template does not configure one.
	maxLength := stepInfo.GetInt("max-length")

	if maxLength <= 0 {
		maxLength = editContentDefaultMaxLength
	}

	// RULE: Clamp to the storage ceiling.  A template must never allow more content than
	// the underlying schema can persist, so an over-large value is capped (not rejected)
	// and the misconfiguration is logged.
	if maxLength > editContentMaxLengthCeiling {
		log.Warn().Str("location", location).Int("configured", maxLength).Int("ceiling", editContentMaxLengthCeiling).Msg("edit-content max-length exceeds storage ceiling; clamping")
		maxLength = editContentMaxLengthCeiling
	}

	// Create the new "edit-content" step
	return EditContent{
		Filename:       first(stepInfo.GetString("file"), stepInfo.GetString("actionId")),
		Fieldname:      first(stepInfo.GetString("field"), "content"),
		Format:         first(stepInfo.GetString("format"), "editorjs"),
		MaxLength:      maxLength,
		RequireContent: stepInfo.GetBool("require-content"),
	}, nil
}

// StepEditContentSchema returns a validating schema for the EditContent step
func StepEditContentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"filename": schema.String{},
			"format": schema.String{
				Required: true,
				Enum: []string{
					"EDITORJS",
					"HTML",
					"MARKDOWN",
					"TEXT",
				},
			},
			"max-length":      schema.Integer{},
			"require-content": schema.Boolean{},
		},
	}
}

// Name returns the name of the step, which is used in debugging.
func (step EditContent) Name() string {
	return "edit-content"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step EditContent) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditContent) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditContent) RequiredRoles() []string {
	return []string{}
}
