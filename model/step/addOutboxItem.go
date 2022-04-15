package step

import (
	"text/template"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// AddOutboxItem represents an action-step that logs activity to a user's outbox
type AddOutboxItem struct {
	Type    string
	Link    string
	Comment *template.Template
}

// NewAddOutboxItem returns a fully populated AddOutboxItem object
func NewAddOutboxItem(stepInfo datatype.Map) (AddOutboxItem, error) {

	commentString := stepInfo.GetString("comment")
	comment, err := template.New("").Parse(commentString)

	if err != nil {
		return AddOutboxItem{}, derp.Wrap(err, "model.step.NewAddOutboxItem", "Invalid 'comment'", commentString)
	}

	return AddOutboxItem{
		Type:    stepInfo.GetString("type"),
		Link:    stepInfo.GetString("link"),
		Comment: comment,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AddOutboxItem) AmStep() {}
