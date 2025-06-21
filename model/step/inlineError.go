package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// InlineError is a Step that displays an "inline failure" message on a form
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

// Name returns the name of the step, which is used in debugging.
func (step InlineError) Name() string {
	return "inline-error"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step InlineError) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step InlineError) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step InlineError) RequiredRoles() []string {
	return []string{}
}
