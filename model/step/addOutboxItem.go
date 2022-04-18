package step

import (
	"text/template"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// AddOutboxItem represents an action-step that logs activity to a user's outbox
type AddOutboxItem struct {
	Label       *template.Template
	Description *template.Template
	Type        string
	Link        string
}

// NewAddOutboxItem returns a fully populated AddOutboxItem object
func NewAddOutboxItem(stepInfo datatype.Map) (AddOutboxItem, error) {

	labelString := stepInfo.GetString("label")
	label, err := template.New("").Parse(labelString)

	if err != nil {
		return AddOutboxItem{}, derp.Wrap(err, "model.step.NewAddOutboxItem", "Invalid 'label'", labelString)
	}

	descriptionString := stepInfo.GetString("description")
	description, err := template.New("").Parse(descriptionString)

	if err != nil {
		return AddOutboxItem{}, derp.Wrap(err, "model.step.NewAddOutboxItem", "Invalid 'description'", descriptionString)
	}

	return AddOutboxItem{
		Label:       label,
		Description: description,
		Type:        stepInfo.GetString("type"),
		Link:        stepInfo.GetString("link"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AddOutboxItem) AmStep() {}