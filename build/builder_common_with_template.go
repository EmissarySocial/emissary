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
	return false
}

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (builder CommonWithTemplate) UserCan(actionID string) bool {

	// Find the action in the Template
	action, exists := builder._template.Actions[actionID]

	if !exists {
		return false
	}

	// Get a list of the valid roles for this action
	allowList := action.AllowList[builder._accessLister.State()]

	// If Anonymous access is allowed, then EVERYONE can perform this action
	if allowList.Anonymous {
		return true
	}

	//////////////////////////////////////////////////////////////////
	// Beyond this point, you must be logged in to perform this action

	// If the user is a domain owner, then they can do anything
	if builder._authorization.DomainOwner {
		return true
	}

	// If "Authenticated" access is allowed, then any LOGGED IN USERS can perform this action
	if allowList.Authenticated {
		if builder._authorization.IsAuthenticated() {
			return true
		}
	}

	// If the allowList allows "author" access, then check to see if the user is the author of this object
	if allowList.Author {
		if builder._accessLister.IsAuthor(builder._authorization.UserID) {
			return true
		}
	}

	// If the allowList allows "myself" access, then check to see if this user is "myself"
	if allowList.Self {
		if builder._accessLister.IsMyself(builder._authorization.UserID) {
			return true
		}
	}

	// Check for group access via the Authorization object
	if len(allowList.GroupRoles) > 0 {

		if builder._authorization.IsAuthenticated() {

			// Map the list of allowed roles to a list of GroupIDs
			groupIDs := builder._accessLister.RolesToGroupIDs(allowList.GroupRoles...)

			// Continue ONLY IF there is at least one are an GroupID that allows access to this action
			if len(groupIDs) > 0 {

				if builder._authorization.IsGroupMember(groupIDs...) {
					return true
				}
			}
		}
	}

	// Query the database to see if this User has purchased any allowed Products...
	if len(allowList.ProductRoles) > 0 {

		// We can check for product ownership ONLY IF there is a valid GuestID
		guestID := builder._authorization.GuestID

		if !guestID.IsZero() {

			// Map the list of allowed roles to a list of ProductIDs
			productIDs := builder._accessLister.RolesToProductIDs(allowList.ProductRoles...)

			// Continue ONLY IF there is at least one ProductID that allows access to this action
			if len(productIDs) > 0 {

				// Count the number of purchases for these Guest/Product combinations
				purchaseService := builder._factory.Purchase()
				count, err := purchaseService.CountByGuestAndProduct(guestID, productIDs...)

				if err != nil {
					derp.Report(derp.Wrap(err, "builder.UserCan", "Unable to count purchases"))
					return false
				}

				// Allow if the guest owns any of the Products
				if count > 0 {
					return true
				}
			}
		}
	}

	// The user does not have permission to perform this action
	return false
}
