package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Delete is a Step that can delete a Stream from the Domain
type Delete struct {
	Title   *template.Template
	Message *template.Template
	Submit  string
	Method  string
}

// NewDelete returns a fully populated Delete object
func NewDelete(stepInfo mapof.Any) (Delete, error) {

	// Validate the step configuration
	if err := StepDeleteSchema().Validate(stepInfo); err != nil {
		return Delete{}, derp.Wrap(err, "model.step.NewDelete", "Invalid step configuration", stepInfo)
	}

	// Create the "title" template
	titleTemplate, err := template.New("").Parse(first(stepInfo.GetString("title"), "Delete '{{.Label}}'?"))

	if err != nil {
		return Delete{}, derp.Wrap(err, "model.step.NewDelete", "Invalid 'title' template", stepInfo)
	}

	// Create the "message" template
	messageTemplate, err := template.New("").Parse(first(stepInfo.GetString("message"), "Are you sure you want to delete {{.Label}}? There is NO UNDO."))

	if err != nil {
		return Delete{}, derp.Wrap(err, "model.step.NewDelete", "Invalid 'message' template", stepInfo)
	}

	return Delete{
		Title:   titleTemplate,
		Message: messageTemplate,
		Submit:  first(stepInfo.GetString("submit"), "Delete"),
		Method:  first(stepInfo.GetString("method"), "both"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Delete) AmStep() {}

// StepDeleteSchema returns a validating schema for the EditContent step
func StepDeleteSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"title":   schema.String{MaxLength: 1024, Format: "text"},
			"message": schema.String{MaxLength: 128, Format: "text"},
			"submit":  schema.String{MaxLength: 32, Format: "text"},
			"method":  schema.String{Enum: []string{"get", "post", "both"}, Default: "both"},
		},
	}
}
