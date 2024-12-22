package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// InlineError is an action-step that displays an "inline failure" message on a form
type InlineError struct {
	Message *template.Template
}

func NewInlineError(stepInfo mapof.Any) (InlineError, error) {

	message, err := template.New("").Funcs(FuncMap()).Parse(stepInfo.GetString("message"))

	if err != nil {
		return InlineError{}, derp.Wrap(err, "model.step.NewInlineError", "Error parsing template")
	}

	return InlineError{
		Message: message,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step InlineError) AmStep() {}
