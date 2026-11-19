package step

import (
	"maps"
	"slices"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// Switch is a Step that renders an expression, then runs the sub-pipeline of the case whose
// key matches it, or the default sub-pipeline when no case matches
type Switch struct {
	Expression *template.Template
	Cases      map[string][]Step
	Default    []Step
}

// NewSwitch returns a fully initialized Switch step, or an error if its configuration is invalid
func NewSwitch(stepInfo mapof.Any, funcMap template.FuncMap) (Switch, error) {

	const location = "model.step.NewSwitch"

	// RULE: expression is required, because an empty one would always run the same case
	expressionText := stepInfo.GetString("expression")

	if expressionText == "" {
		return Switch{}, derp.Internal(location, "Missing 'expression'")
	}

	// Parse "expression" property
	expression, err := template.New("").Funcs(funcMap).Parse(expressionText)

	if err != nil {
		return Switch{}, derp.Wrap(err, location, "Invalid 'expression'")
	}

	// RULE: cases must be a map with at least one case, so a misspelled key cannot pass as empty
	caseInfo, isMap := switchMap(stepInfo["cases"])

	if !isMap {
		return Switch{}, derp.Internal(location, "'cases' must map each value to a step or a list of steps")
	}

	if len(caseInfo) == 0 {
		return Switch{}, derp.Internal(location, "'cases' must contain at least one case")
	}

	// Parse each case's sub-pipeline
	cases := make(map[string][]Step, len(caseInfo))

	for key, value := range caseInfo {

		steps, err := switchSteps(value, funcMap)

		if err != nil {
			return Switch{}, derp.Wrap(err, location, "Invalid case", key)
		}

		cases[key] = steps
	}

	// Parse the optional "default" property
	defaultSteps := make([]Step, 0)

	if value := stepInfo["default"]; value != nil {

		defaultSteps, err = switchSteps(value, funcMap)

		if err != nil {
			return Switch{}, derp.Wrap(err, location, "Invalid 'default'")
		}
	}

	return Switch{
		Expression: expression,
		Cases:      cases,
		Default:    defaultSteps,
	}, nil
}

// switchMap returns the value as a mapof.Any, and TRUE if it is a map
func switchMap(value any) (mapof.Any, bool) {

	switch typed := value.(type) {

	case mapof.Any:
		return typed, true

	case map[string]any:
		return mapof.Any(typed), true
	}

	return nil, false
}

// switchSteps parses one branch of a Switch step, which is either a single step or a list of steps
func switchSteps(value any, funcMap template.FuncMap) ([]Step, error) {

	const location = "model.step.switchSteps"

	// A single step is a pipeline of one
	if stepInfo, isMap := switchMap(value); isMap {
		return NewPipeline([]mapof.Any{stepInfo}, funcMap)
	}

	// A list of steps must hold nothing but maps
	switch typed := value.(type) {

	case []mapof.Any:
		return NewPipeline(typed, funcMap)

	case []any:
		stepInfo, isSliceOfMap := convert.SliceOfMapOk(typed)

		if !isSliceOfMap {
			return nil, derp.Internal(location, "Each item in a list of steps must be a step")
		}

		return NewPipeline(stepInfo, funcMap)
	}

	return nil, derp.Internal(location, "Value must be a step or a list of steps")
}

// Name returns the name of the step, which is used in debugging.
func (step Switch) Name() string {
	return "switch"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step Switch) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Switch) RequiredStates() []string {

	result := requiredStates(step.Default...)

	for _, key := range slices.Sorted(maps.Keys(step.Cases)) {
		result = append(result, requiredStates(step.Cases[key]...)...)
	}

	return result
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Switch) RequiredRoles() []string {

	result := requiredRoles(step.Default...)

	for _, key := range slices.Sorted(maps.Keys(step.Cases)) {
		result = append(result, requiredRoles(step.Cases[key]...)...)
	}

	return result
}
