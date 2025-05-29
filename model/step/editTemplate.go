package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// EditTemplate is a Step that lets users edit an object's template(s)
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
			return EditTemplate{}, derp.BadRequestError(location, "Invalid value.  Only 'templateId', 'inboxTemplate', and 'outboxTemplate' are allowed", key)
		}
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step EditTemplate) Name() string {
	return "edit-template"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step EditTemplate) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step EditTemplate) RequiredRoles() []string {
	return []string{}
}
