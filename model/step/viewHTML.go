package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// ViewHTML is a Step that can build a Stream into HTML
type ViewHTML struct {
	File         string
	Method       string
	CacheControl string
	AsFullPage   bool
}

// NewViewHTML generates a fully initialized ViewHTML step.
func NewViewHTML(stepInfo mapof.Any) (ViewHTML, error) {

	method, err := parseMethod(stepInfo, "get")

	if err != nil {
		return ViewHTML{}, derp.Wrap(err, "model.step.NewViewHTML", "Invalid 'method'", stepInfo)
	}

	return ViewHTML{
		File: stepInfo.GetString("file"),

		Method: method,

		// Left empty on purpose.  The default lives in the build step, beside the headers it guards,
		// so that a ViewHTML assembled any other way still gets a safe policy.
		CacheControl: stepInfo.GetString("cache-control"),

		AsFullPage: stepInfo.GetBool("as-full-page"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ViewHTML) Name() string {
	return "view-html"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ViewHTML) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ViewHTML) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ViewHTML) RequiredRoles() []string {
	return []string{}
}
