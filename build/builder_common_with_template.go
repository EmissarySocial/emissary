package build

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

type CommonWithTemplate struct {
	_actionID string
	_action   model.Action
	_template model.Template

	Common
}

func NewCommonWithTemplate(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, actionID string) (CommonWithTemplate, error) {

	const location = "build.NewCommonWithTemplate"

	// Locate the Action inside the Template
	action, ok := template.Action(actionID)

	if !ok {
		return CommonWithTemplate{}, derp.BadRequestError(location, "Action is not valid", actionID)
	}

	// Return the CommonWithTemplate builder
	return CommonWithTemplate{
		_actionID: actionID,
		_action:   action,
		_template: template,
		Common:    NewCommon(factory, request, response),
	}, nil
}

/******************************************
 * Builder Interface
 ******************************************/

func (builder CommonWithTemplate) actions() map[string]model.Action {
	return builder._template.Actions
}

// Action returns the model.Action configured into this builder
func (builder CommonWithTemplate) action() model.Action {
	return builder._action
}

func (builder CommonWithTemplate) actionID() string {
	return builder._actionID
}

// template returns the model.Template associated with this Builder
func (builder CommonWithTemplate) template() model.Template {
	return builder._template
}

// execute writes the named HTML template into a writer using the provided data
func (builder CommonWithTemplate) execute(wr io.Writer, name string, data any) error {
	return builder._template.HTMLTemplate.ExecuteTemplate(wr, name, data)
}
