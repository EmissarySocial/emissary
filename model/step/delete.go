package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// Delete represents an action-step that can delete a Stream from the Domain
type Delete struct {
	Title   *template.Template
	Message *template.Template
	Submit  string
}

// NewDelete returns a fully populated Delete object
func NewDelete(stepInfo mapof.Any) (Delete, error) {

	titleTemplate, err := template.New("").Parse(first.String(stepInfo.GetString("title"), "Delete '{{.Label}}'?"))

	if err != nil {
		return Delete{}, derp.Wrap(err, "model.step.NewDelete", "Invalid 'title' template", stepInfo)
	}

	messageTemplate, err := template.New("").Parse(first.String(stepInfo.GetString("message"), "Are you sure you want to delete {{.Label}}? There is NO UNDO."))

	if err != nil {
		return Delete{}, derp.Wrap(err, "model.step.NewDelete", "Invalid 'message' template", stepInfo)
	}

	return Delete{
		Title:   titleTemplate,
		Message: messageTemplate,
		Submit:  first.String(stepInfo.GetString("submit"), "Delete"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Delete) AmStep() {}
