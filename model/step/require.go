package step

import (
	"strings"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Require is a Step that halts the pipeline with a 400 Bad Request error unless its condition is true
type Require struct {
	Condition *template.Template
}

// NewRequire returns a fully initialized Require step, or an error if its configuration is invalid
func NewRequire(stepInfo mapof.Any, funcMap template.FuncMap) (Require, error) {

	const location = "model.step.NewRequire"

	// RULE: condition is required, because an empty one would refuse every request
	conditionText := stepInfo.GetString("condition")

	if strings.TrimSpace(conditionText) == "" {
		return Require{}, derp.Internal(location, "Missing 'condition'")
	}

	// Parse "condition" property
	condition, err := template.New("").Funcs(funcMap).Parse(conditionText)

	if err != nil {
		return Require{}, derp.Wrap(err, location, "Invalid 'condition'")
	}

	return Require{
		Condition: condition,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Require) Name() string {
	return "require"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step Require) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Require) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Require) RequiredRoles() []string {
	return []string{}
}
