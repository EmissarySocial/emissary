package step

import (
	"html/template"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// RedirectTo is a Step that forwards the user to a new page.
type RedirectTo struct {
	StatusCode int
	URL        *template.Template
}

// NewRedirectTo returns a fully initialized RedirectTo object
func NewRedirectTo(stepInfo mapof.Any) (RedirectTo, error) {

	const location = "model.step.NewRedirectTo"

	url, err := template.New("").Parse(stepInfo.GetString("url"))

	if err != nil {
		return RedirectTo{}, derp.Wrap(err, location, "Invalid 'url' template", stepInfo)
	}

	return RedirectTo{
		StatusCode: first(stepInfo.GetInt("status"), http.StatusTemporaryRedirect),
		URL:        url,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step RedirectTo) AmStep() {}
