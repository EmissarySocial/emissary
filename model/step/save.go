package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Save represents an action-step that can save changes to any object
type Save struct {
	Comment *template.Template
}

// NewSave returns a fully initialized Save object
func NewSave(stepInfo mapof.Any) (Save, error) {

	comment, err := template.New("").Parse(stepInfo.GetString("comment"))

	if err != nil {
		return Save{}, derp.Wrap(err, "model.step.NewSave", "Error parsing comment template")
	}

	return Save{
		Comment: comment,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Save) AmStep() {}
