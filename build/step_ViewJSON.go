package build

import (
	"encoding/json"
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/translate"
)

// StepViewJSON is a Step that writes a JSON document, either from one template expression (Value)
// or from a rules pipeline (Rules) applied to the builder's object
type StepViewJSON struct {
	Value       *template.Template
	Schema      schema.Schema
	Rules       translate.Pipeline
	ContentType string
}

// Get writes the JSON document to the context
func (step StepViewJSON) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepViewJSON.Get"

	// Build the document from the rules
	if step.Value == nil {

		result, err := step.buildDocument(builder.schema(), builder.object())

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Applying rules"))
		}

		if err := json.NewEncoder(buffer).Encode(result); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Encoding JSON"))
		}

		return Continue().AsFullPage().WithContentType(step.ContentType)
	}

	// Otherwise, render the "value" expression
	if err := step.Value.Execute(buffer, builder); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Executing template"))
	}

	return Continue().AsFullPage().WithContentType(step.ContentType)
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepViewJSON) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return Continue()
}

// buildDocument returns a new document, written by the rules from the object read through its schema
func (step StepViewJSON) buildDocument(objectSchema schema.Schema, object any) (mapof.Any, error) {

	const location = "build.StepViewJSON.buildDocument"

	// RULE: Every request writes into its own map, because the step is shared by every request
	result := mapof.Any{}

	if err := step.Rules.Execute(objectSchema, object, step.Schema, &result); err != nil {
		return nil, derp.Wrap(err, location, "Executing rules")
	}

	return result, nil
}
