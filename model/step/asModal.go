package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// AsModal is a Step that can update the data.DataMap custom data stored in a Stream
type AsModal struct {
	SubSteps   []Step
	Options    []*template.Template // Each option, like "class:large", executed against the builder
	Background string
}

// NewAsModal returns a fully initialized AsModal object
func NewAsModal(stepInfo mapof.Any, funcMap template.FuncMap) (AsModal, error) {

	const location = "model.step.NewAsModal"

	// Parse the steps that build the modal's content
	subSteps, err := NewPipeline(stepInfo.GetSliceOfMap("steps"), funcMap)

	if err != nil {
		return AsModal{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	// Parse each option as a template
	options := stepInfo.GetSliceOfString("options")
	optionTemplates := make([]*template.Template, len(options))

	for index, option := range options {

		optionTemplate, err := template.New("option").Funcs(funcMap).Parse(option)

		if err != nil {
			return AsModal{}, derp.Wrap(err, location, "Invalid 'options'", stepInfo)
		}

		optionTemplates[index] = optionTemplate
	}

	return AsModal{
		SubSteps:   subSteps,
		Options:    optionTemplates,
		Background: stepInfo.GetString("background"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step AsModal) Name() string {
	return "as-modal"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step AsModal) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step AsModal) RequiredStates() []string {
	return requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined by any Template that uses this Step
func (step AsModal) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
