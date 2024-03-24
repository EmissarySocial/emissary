package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SetTemplate is a headless step that updates the Template(s) used by a Stream or User
type SetTemplate struct {
	Paths map[string]*template.Template
}

func NewSetTemplate(stepInfo mapof.Any) (SetTemplate, error) {

	const location = "model.step.SetTemplate.NewSetTemplate"

	result := SetTemplate{
		Paths: make(map[string]*template.Template, 0),
	}

	for key := range stepInfo {

		switch key {

		case "do":
			// NO OP

		case "templateId", "inboxTemplate", "outboxTemplate":
			t, err := template.New(key).Parse(stepInfo.GetString(key))

			if err != nil {
				return SetTemplate{}, derp.Wrap(err, location, "Error parsing template", key)
			}

			result.Paths[key] = t

		default:
			return SetTemplate{}, derp.New(derp.CodeBadRequestError, location, "Invalid value.  Only 'templateId', 'inboxTemplate', and 'outboxTemplate' are allowed", key)
		}
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetTemplate) AmStep() {}
