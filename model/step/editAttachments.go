package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// EditAttachments is a Step that draws the files in one category as tiles, and uploads,
// deletes, and sorts them in place, behind an html-remote form element
type EditAttachments struct {
	Category   string   // Category of the files this step manages
	Fieldname  string   // Form field that posts the files, and whose ".order" field posts their order (Default: the category)
	AcceptType string   // Mime Type(s) to accept (e.g. "image/*")
	Minimum    int      // Fewest files the category may hold (Default: 0)
	Maximum    int      // Most files the category may hold (Default: 1)
	RuleHeight int      // Fixed height for all downloads
	RuleWidth  int      // Fixed width for all downloads
	RuleTypes  []string // Allowed extensions.  The first value is used as the default.
}

// NewEditAttachments returns a fully parsed EditAttachments object
func NewEditAttachments(stepInfo mapof.Any) (EditAttachments, error) {

	const location = "step.NewEditAttachments"

	// RULE: The category names the files this step manages, so the step cannot run without one
	category := stepInfo.GetString("category")

	if category == "" {
		return EditAttachments{}, derp.BadRequest(location, "The 'category' attribute is required")
	}

	maximum := max(stepInfo.GetInt("maximum"), 1)
	minimum := max(stepInfo.GetInt("minimum"), 0)

	// RULE: A minimum above the maximum could never be met
	if minimum > maximum {
		return EditAttachments{}, derp.BadRequest(location, "The 'minimum' attribute cannot exceed the 'maximum'", minimum, maximum)
	}

	rules := stepInfo.GetMap("rules")

	return EditAttachments{
		Category:   category,
		Fieldname:  first(stepInfo.GetString("fieldname"), category),
		AcceptType: stepInfo.GetString("accept-type"),
		Minimum:    minimum,
		Maximum:    maximum,
		RuleHeight: rules.GetInt("height"),
		RuleWidth:  rules.GetInt("width"),
		RuleTypes:  rules.GetSliceOfString("types"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step EditAttachments) Name() string {
	return "edit-attachments"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step EditAttachments) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditAttachments) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditAttachments) RequiredRoles() []string {
	return []string{}
}
