package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/maps"
)

// TriggerEvent represents an action-step that forwards the user to a new page.
type TriggerEvent struct {
	Event string
	Data  *template.Template
}

// NewTriggerEvent returns a fully initialized TriggerEvent object
func NewTriggerEvent(stepInfo maps.Map) (TriggerEvent, error) {

	dataString := stepInfo.GetString("data")
	data, err := template.New("").Parse(dataString)

	if err != nil {
		return TriggerEvent{}, derp.Wrap(err, "model.step.NewTriggerEvent", "Invalid data template", dataString)
	}

	return TriggerEvent{
		Event: stepInfo.GetString("event"),
		Data:  data,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step TriggerEvent) AmStep() {}
