package step

import (
	"html/template"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// ForwardTo is a Step that forwards the user to a new page.
type ForwardTo struct {
	URL    *template.Template
	Method string
}

// NewForwardTo returns a fully initialized ForwardTo object
func NewForwardTo(stepInfo mapof.Any) (ForwardTo, error) {

	const location = "model.step.NewForwardTo"

	url, err := template.New("").Parse(stepInfo.GetString("url"))

	if err != nil {
		return ForwardTo{}, derp.Wrap(err, location, "Invalid 'url' template", stepInfo)
	}

	return ForwardTo{
		URL:    url,
		Method: first(strings.ToLower(stepInfo.GetString("method")), "post"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ForwardTo) Name() string {
	return "forward-to"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ForwardTo) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ForwardTo) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ForwardTo) RequiredRoles() []string {
	return []string{}
}
