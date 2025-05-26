package build

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

type CommonWithTemplate struct {
	_actionID     string
	_action       model.Action
	_template     model.Template
	_accessLister model.AccessLister

	Common
}

func NewCommonWithTemplate(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, accessLister model.AccessLister, actionID string) (CommonWithTemplate, error) {

	const location = "build.NewCommonWithTemplate"

	// Locate the Action inside the Template
	action, ok := template.Action(actionID)

	if !ok {
		return CommonWithTemplate{}, derp.BadRequestError(location, "Action is not valid", actionID)
	}

	// Create the CommonWithTemplate builder
	result := CommonWithTemplate{
		_actionID:     actionID,
		_action:       action,
		_template:     template,
		_accessLister: accessLister,
		Common:        NewCommon(factory, request, response),
	}

	// Calculate permissions...
	// what can be cached here, and what needs to be recalculated [[SEVERAL TIMES PER REQUEST..]]

	return result, nil
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

/******************************************
 * Access Permissions Methods
 ******************************************/

// AuthorInGroup returns TRUE if the Author/AttributedTo is a member of the specified group
func (w Stream) AuthorInGroup(string) bool {
	return false
}

// UserInGroup returns TRUE if the user is a member of the specified group
func (w Stream) UserInGroup(groupID string) bool {
	return false
}

// UserHasRole returns TRUE if the user has privileges for the specified role
func (w Stream) UserHasRole(role string) bool {

	const location = "builder.Stream.UserHasRole"

	// Use the Permission service to check if the user has the specified role
	permissionService := w._factory.Permission()
	hasRole, err := permissionService.UserHasRole(&w._authorization, w._accessLister, role)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to check user roles"))
		return false
	}

	return hasRole
}

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (builder CommonWithTemplate) UserCan(actionID string) bool {

	permissionService := builder._factory.Permission()
	result, err := permissionService.UserCan(&builder._authorization, &builder._template, builder._accessLister, actionID)

	if err != nil {
		derp.Report(derp.Wrap(err, "builder.UserCan", "Unable to check permissions"))
		return false
	}

	return result
}
