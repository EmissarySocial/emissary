package step

import (
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
)

// The `max-length` step setting is configured in KILOBYTES for developer ergonomics
// (e.g. `max-length:64`), then converted to runes internally.  One "kilobyte" here is
// 1024 characters.
const (
	// editContentDefaultMaxLengthKB is the per-step content limit (in KB) applied when a
	// template does not specify its own `max-length`.  It is generous enough for long-form
	// articles while still rejecting the multi-megabyte bodies that a storage-exhaustion
	// attack relies on.
	editContentDefaultMaxLengthKB = 64 // 64 KB

	// editContentMaxLengthCeilingKB is the largest `max-length` (in KB) a template may
	// configure.  A template cannot allow more content than the system will actually store,
	// so this is kept in sync with model.ContentMaxLength (the schema-level cap on
	// `content.raw`/`content.html`).  It is duplicated here rather than imported because
	// model imports model/step, so model/step cannot import model without creating a cycle.
	editContentMaxLengthCeilingKB = 1024 // 1024 KB == model.ContentMaxLength (1 MiB)

	// runesPerKilobyte converts the KB-denominated `max-length` setting into the rune count
	// used for enforcement.
	runesPerKilobyte = 1024
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
// `max-length` setting is expressed in kilobytes and converted to runes here.
func NewEditContent(stepInfo mapof.Any) (EditContent, error) {

	const location = "model.step.NewEditContent"

	// Validate the step configuration
	if _, err := schema.New(StepEditContentSchema()).Validate(stepInfo); err != nil {
		return EditContent{}, err
	}

	// RULE: Apply the default limit when a template does not configure one.
	maxLengthKB := stepInfo.GetInt("max-length")

	if maxLengthKB <= 0 {
		maxLengthKB = editContentDefaultMaxLengthKB
	}

	// RULE: Clamp to the storage ceiling.  A template must never allow more content than
	// the underlying schema can persist, so an over-large value is capped (not rejected)
	// and the misconfiguration is logged.
	if maxLengthKB > editContentMaxLengthCeilingKB {
		log.Warn().Str("location", location).Int("configuredKB", maxLengthKB).Int("ceilingKB", editContentMaxLengthCeilingKB).Msg("edit-content max-length exceeds storage ceiling; clamping")
		maxLengthKB = editContentMaxLengthCeilingKB
	}

	// Create the new "edit-content" step
	return EditContent{
		Filename:       first(stepInfo.GetString("file"), stepInfo.GetString("actionId")),
		Fieldname:      first(stepInfo.GetString("field"), "content"),
		Format:         first(stepInfo.GetString("format"), "editorjs"),
		MaxLength:      maxLengthKB * runesPerKilobyte,
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
