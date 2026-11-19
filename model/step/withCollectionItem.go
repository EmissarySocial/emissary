package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithCollectionItem is a Step that runs a list of sub-steps on one CollectionItem that the current User owns.
type WithCollectionItem struct {
	Type     string
	SubSteps []Step
}

// NewWithCollectionItem returns a fully initialized WithCollectionItem object
func NewWithCollectionItem(stepInfo mapof.Any, funcMap template.FuncMap) (WithCollectionItem, error) {

	const location = "model.step.NewWithCollectionItem"

	// RULE: The step must name the one collection type that it works on
	collectionType := stepInfo.GetString("type")

	if collectionType == "" {
		return WithCollectionItem{}, derp.Validation("Step requires a 'type'", stepInfo, derp.WithLocation(location))
	}

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]), funcMap)

	if err != nil {
		return WithCollectionItem{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithCollectionItem{
		Type:     collectionType,
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithCollectionItem) Name() string {
	return "with-collection-item"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithCollectionItem) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithCollectionItem) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithCollectionItem) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
