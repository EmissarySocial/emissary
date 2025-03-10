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

// AmStep is here only to verify that this struct is a build pipeline step
func (step SearchIndex) AmStep() {}
