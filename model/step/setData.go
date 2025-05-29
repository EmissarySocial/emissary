package step

import (
	"text/template"

	"github.com/benpate/rosetta/mapof"
)

// SetData is a Step that can update the custom data stored in a Stream
type SetData struct {
	FromURL  []string                      // List of paths to pull from URL data
	FromForm []string                      // List of paths to pull from Form data
	Values   map[string]*template.Template // values to set directly into the object
	Defaults mapof.Any                     // values to set into the object IFF they are currently empty.
}

// NewSetData returns a fully initialized SetData object
func NewSetData(stepInfo mapof.Any) (SetData, error) {

	// Read all value templates from the stepInfo map
	valuesMap := stepInfo.GetMap("values")
	values := make(map[string]*template.Template)

	// Parse each template
	for key := range stepInfo.GetMap("values") {
		valueTemplate, err := template.New(key).Funcs(FuncMap()).Parse(valuesMap.GetString(key))
		if err != nil {
			return SetData{}, err
		}
		values[key] = valueTemplate
	}

	return SetData{
		FromURL:  stepInfo.GetSliceOfString("from-url"),
		FromForm: stepInfo.GetSliceOfString("from-form"),
		Values:   values,
		Defaults: stepInfo.GetMap("defaults"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetData) Name() string {
	return "set-data"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetData) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetData) RequiredRoles() []string {
	return []string{}
}
