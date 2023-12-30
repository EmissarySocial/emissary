package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// TriggerEvent represents an action-step that forwards the user to a new page.
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

// AmStep is here only to verify that this struct is a render pipeline step
func (step TriggerEvent) AmStep() {}
