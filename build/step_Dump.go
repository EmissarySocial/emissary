package build

import (
	"io"
	"text/template"

	"github.com/davecgh/go-spew/spew"
)

// StepDump is a Step that can delete a Stream from the Domain
type StepDump struct {
	Value *template.Template
}

// Get displays a customizable confirmation form for the delete
func (step StepDump) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if value := executeTemplate(step.Value, builder); value == "" {
		spew.Dump(builder.object())
	} else {
		spew.Dump(value)
	}

	return Continue()
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDump) Post(builder Builder, _ io.Writer) PipelineBehavior {

	if value := executeTemplate(step.Value, builder); value == "" {
		spew.Dump(builder.object())
	} else {
		spew.Dump(value)
	}

	return Continue()
}
