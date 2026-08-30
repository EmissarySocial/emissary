package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// ReadForm is a Step that reads named fields from a form POST into the Builder's
// temporary data scope, where later steps can render them
type ReadForm struct {
	Schema schema.Schema // Describes and bounds every field this step will accept
}

// NewReadForm returns a fully initialized ReadForm object
func NewReadForm(stepInfo mapof.Any) (ReadForm, error) {

	const location = "model.step.NewReadForm"

	// RULE: the schema is what bounds visitor input.  A step with no schema would accept any
	// field, at any length, which is how an unbounded form field becomes an unbounded header.
	schemaMap := stepInfo.GetMap("schema")

	if len(schemaMap) == 0 {
		return ReadForm{}, derp.BadRequest(location, "Step 'read-form' requires a 'schema' that bounds every field it accepts")
	}

	result := ReadForm{}

	if err := result.Schema.UnmarshalMap(schemaMap); err != nil {
		return ReadForm{}, derp.Wrap(err, location, "Invalid 'schema'", schemaMap)
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ReadForm) Name() string {
	return "read-form"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ReadForm) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ReadForm) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ReadForm) RequiredRoles() []string {
	return []string{}
}
