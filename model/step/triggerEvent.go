package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// TriggerEvent is a Step that forwards the user to a new page.
type TriggerEvent struct {
	Event string
	Value *template.Template
}

// NewTriggerEvent returns a fully initialized TriggerEvent object
func NewTriggerEvent(stepInfo mapof.Any) (TriggerEvent, error) {

	value, err := template.New("").Funcs(FuncMap()).Parse(stepInfo.GetString("value"))

	if err != nil {
		return TriggerEvent{}, derp.Wrap(err, "model.step.NewTriggerEvent", "Error parsing template")
	}

	return TriggerEvent{
		Event: stepInfo.GetString("event"),
		Value: value,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step TriggerEvent) Name() string {
	return "trigger-event"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step TriggerEvent) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step TriggerEvent) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step TriggerEvent) RequiredRoles() []string {
	return []string{}
}
