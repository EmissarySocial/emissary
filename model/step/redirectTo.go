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

// Name returns the name of the step, which is used in debugging.
func (step RedirectTo) Name() string {
	return "redirect-to"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step RedirectTo) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step RedirectTo) RequiredRoles() []string {
	return []string{}
}
