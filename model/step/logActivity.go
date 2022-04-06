package step

import (
	"text/template"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// LogActivity represents an action-step that can delete a Stream from the Domain
type LogActivity struct {
	Type      string
	Link      string
	Container string
	Comment   *template.Template
}

// NewLogActivity returns a fully populated LogActivity object
func NewLogActivity(stepInfo datatype.Map) (LogActivity, error) {

	commentString := stepInfo.GetString("comment")
	comment, err := template.New("").Parse(commentString)

	if err != nil {
		return LogActivity{}, derp.Wrap(err, "model.step.NewLogActivity", "Invalid 'comment'", commentString)
	}

	return LogActivity{
		Type:      stepInfo.GetString("type"),
		Link:      stepInfo.GetString("link"),
		Container: stepInfo.GetString("container"),
		Comment:   comment,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step LogActivity) AmStep() {}
