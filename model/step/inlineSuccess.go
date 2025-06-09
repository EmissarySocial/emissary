package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// InlineSuccess is a Step that displays an "inline success" message on a form
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

// Name returns the name of the step, which is used in debugging.
func (step InlineSuccess) Name() string {
	return "inline-success"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step InlineSuccess) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step InlineSuccess) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step InlineSuccess) RequiredRoles() []string {
	return []string{}
}
