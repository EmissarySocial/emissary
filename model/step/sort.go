package step

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Sort is a Step that can update multiple records at once
type Sort struct {
	Model     string
	Keys      string
	Values    string
	Direction string
	Message   string
}

// NewSort returns a fully initialized Sort step, or an error if its configuration is invalid
func NewSort(stepInfo mapof.Any) (Sort, error) {

	const location = "step.NewSort"

	// RULE: Direction must be "asc" (the default) or "desc"
	direction := strings.ToLower(first(stepInfo.GetString("direction"), "asc"))

	if direction != "asc" && direction != "desc" {
		return Sort{}, derp.BadRequest(location, "Direction must be 'asc' or 'desc'", direction)
	}

	return Sort{
		Model:     stepInfo.GetString("model"),
		Keys:      first(stepInfo.GetString("keys"), "_id"),
		Values:    first(stepInfo.GetString("values"), "rank"),
		Direction: direction,
		Message:   stepInfo.GetString("message"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Sort) Name() string {
	return "sort"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step Sort) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Sort) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Sort) RequiredRoles() []string {
	return []string{}
}
