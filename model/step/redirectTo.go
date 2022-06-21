package step

import (
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/maps"
)

// RedirectTo represents an action-step that forwards the user to a new page.
type RedirectTo struct {
	URL *template.Template
}

// NewRedirectTo returns a fully initialized RedirectTo object
func NewRedirectTo(stepInfo maps.Map) (RedirectTo, error) {

	const location = "model.step.NewRedirectTo"

	url, err := template.New("").Parse(stepInfo.GetString("url"))

	if err != nil {
		return RedirectTo{}, derp.Wrap(err, location, "Invalid 'url' template", stepInfo)
	}

	return RedirectTo{
		URL: url,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step RedirectTo) AmStep() {}
