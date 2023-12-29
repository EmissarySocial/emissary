package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// InlineSuccess represents an action-step that displays an "inline success" message on a form
type InlineSuccess struct {
	Message *template.Template
}

func NewInlineSuccess(stepInfo mapof.Any) (InlineSuccess, error) {

	message, err := template.New("").Funcs(FuncMap()).Parse(stepInfo.GetString("message"))

	if err != nil {
		return InlineSuccess{}, derp.Wrap(err, "model.step.NewInlineSuccess", "Error parsing template")
	}

	return InlineSuccess{
		Message: message,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step InlineSuccess) AmStep() {}
