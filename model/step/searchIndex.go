package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SearchIndex contains the configuration data for a modal that lets administrators manage connections to external servers.
type SearchIndex struct {
	If *template.Template
}

func NewSearchIndex(stepInfo mapof.Any) (SearchIndex, error) {

	// Default "if" condition to "true" if none is provided
	ifString := first(stepInfo.GetString("if"), "true")
	ifTemplate, err := template.New("").Parse(ifString)

	if err != nil {
		return SearchIndex{}, derp.Wrap(err, "step.NewSearchIndex", "Error parsing `if` template")
	}

	// Create the SearchIndex value
	result := SearchIndex{
		If: ifTemplate,
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SearchIndex) Name() string {
	return "search-index"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SearchIndex) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SearchIndex) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SearchIndex) RequiredRoles() []string {
	return []string{}
}
