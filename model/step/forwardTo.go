package step

import (
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// ForwardTo is a Step that forwards the user to a new page.
type ForwardTo struct {
	URL *template.Template
}

// NewForwardTo returns a fully initialized ForwardTo object
func NewForwardTo(stepInfo mapof.Any) (ForwardTo, error) {

	const location = "model.step.NewForwardTo"

	url, err := template.New("").Parse(stepInfo.GetString("url"))

	if err != nil {
		return ForwardTo{}, derp.Wrap(err, location, "Invalid 'url' template", stepInfo)
	}

	return ForwardTo{
		URL: url,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step ForwardTo) AmStep() {}
