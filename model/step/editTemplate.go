package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// EditTemplate is an action-step that lets users edit an object's template(s)
type EditTemplate struct {
	Title string
	Paths []string
}

func NewEditTemplate(stepInfo mapof.Any) (EditTemplate, error) {

	const location = "model.step.EditTemplate.NewEditTemplate"

	result := EditTemplate{
		Title: stepInfo.GetString("title"),
		Paths: make([]string, 0),
	}

	for key := range stepInfo {

		switch key {

		case "do", "title":
			// NO OP

		case "templateId", "inboxTemplate", "outboxTemplate":
			if stepInfo.GetBool(key) {
				result.Paths = append(result.Paths, key)
			}

		default:
			return EditTemplate{}, derp.New(derp.CodeBadRequestError, location, "Invalid value.  Only 'templateId', 'inboxTemplate', and 'outboxTemplate' are allowed", key)
		}
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step EditTemplate) AmStep() {}
