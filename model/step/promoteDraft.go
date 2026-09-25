package step

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// StreamPromoteDraft represents a pipeline-step that can copy the Container from a StreamDraft into its corresponding Stream
type StreamPromoteDraft struct {
	StateID string
	Omit    sliceof.String
}

// promoteDraftFields names every Stream property that `promote-draft` copies out of the draft, and
// is therefore the complete set of names that `omit` accepts.
//
// RULE: nothing links this list to the assignments in service.StreamDraft.Promote, so the two are
// changed together by hand.  A property missing from this list cannot be omitted even though the
// copy still happens; a property listed here that is no longer assigned makes `omit` report success
// for a field it does not govern.
var promoteDraftFields = sliceof.String{
	"url",
	"token",
	"label",
	"summary",
	"content",
	"iconUrl",
	"icon",
	"widgets",
	"data",
	"attributedTo",
	"inReplyTo",
}

// NewStreamPromoteDraft returns a fully initialized StreamPromoteDraft step, or an error if its configuration is invalid
func NewStreamPromoteDraft(stepInfo mapof.Any) (StreamPromoteDraft, error) {

	const location = "step.NewStreamPromoteDraft"

	omit := sliceof.String(stepInfo.GetSliceOfString("omit"))

	// RULE: a name this step does not recognize leaves its property COPIED -- which is the exact
	// mistake `omit` exists to prevent, and a promote that quietly reverts a page reports nothing
	// at all.  So a misspelling fails the Template at load instead of at the next promote.
	for _, name := range omit {

		if strings.Contains(name, ".") {
			return StreamPromoteDraft{}, derp.Internal(location, "Omit cannot name a nested value.  Name the whole property instead", name)
		}

		if promoteDraftFields.NotContains(name) {
			return StreamPromoteDraft{}, derp.Internal(location, "Unknown property in 'omit'", name, "valid properties: "+strings.Join(promoteDraftFields, ", "))
		}
	}

	return StreamPromoteDraft{
		StateID: first(stepInfo.GetString("state"), "published"),
		Omit:    omit,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step StreamPromoteDraft) Name() string {
	return "promote-draft"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step StreamPromoteDraft) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step StreamPromoteDraft) RequiredStates() []string {
	return []string{step.StateID}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step StreamPromoteDraft) RequiredRoles() []string {
	return []string{}
}
