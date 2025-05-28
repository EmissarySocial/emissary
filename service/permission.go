package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

type Permission struct {
	identityService  *Identity
	privilegeService *Privilege
}

func NewPermission() Permission {
	return Permission{}
}

func (service *Permission) Refresh(identityService *Identity, privilegeService *Privilege) {
	service.identityService = identityService
	service.privilegeService = privilegeService
}

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (service *Permission) UserCan(authorization *model.Authorization, template *model.Template, accessLister model.AccessLister, actionID string) (bool, error) {

	const location = "service.Permission.UserCan"

	// Find the action in the Template
	action, exists := template.Actions[actionID]

	if !exists {
		return false, nil
	}

	// Get a list of the valid roles for this action
	allowList := action.AllowList[accessLister.State()]

	// If Anonymous access is allowed, then EVERYONE can perform this action
	if allowList.Anonymous {
		return true, nil
	}

	//////////////////////////////////////////////////////////////////
	// Beyond this point, you must be logged in to perform this action

	// If the user is a domain owner, then they can do anything
	if authorization.DomainOwner {
		return true, nil
	}

	// If "Authenticated" access is allowed, then any LOGGED IN USERS can perform this action
	if allowList.Authenticated {
		if authorization.IsAuthenticated() {
			return true, nil
		}
	}

	// If the allowList allows "author" access, then check to see if the user is the author of this object
	if allowList.Author {
		if accessLister.IsAuthor(authorization.UserID) {
			return true, nil
		}
	}

	// If the allowList allows "myself" access, then check to see if this user is "myself"
	if allowList.Self {
		if accessLister.IsMyself(authorization.UserID) {
			return true, nil
		}
	}

	// Check for group access via the Authorization object
	if len(allowList.GroupRoles) > 0 {

		if authorization.IsAuthenticated() {

			// Map the list of allowed roles to a list of GroupIDs
			groupIDs := accessLister.RolesToGroupIDs(allowList.GroupRoles...)

			// Continue ONLY IF there is at least one are an GroupID that allows access to this action
			if len(groupIDs) > 0 {

				if authorization.IsGroupMember(groupIDs...) {
					return true, nil
				}
			}
		}
	}

	// Query the database to see if this User has purchased any allowed Products...
	if len(allowList.ProductRoles) > 0 {

		// We can check for product ownership ONLY IF there is a valid IdentityID
		identityID := authorization.IdentityID

		if !identityID.IsZero() {

			allowed, err := service.UserHasRole(authorization, accessLister, allowList.ProductRoles...)

			if err != nil {
				return false, derp.Wrap(err, location, "Unable to check user roles")
			}

			if allowed {
				return true, nil
			}
		}
	}

	// The user does not have permission to perform this action
	return false, nil
}

// UserHasRole returns TRUE if the user has privileges for the specified role
func (service *Permission) UserHasRole(authorization *model.Authorization, accessLister model.AccessLister, roles ...string) (bool, error) {

	const location = "service.Permission.UserHasRole"

	// Locate the authorized Identity
	identity := model.NewIdentity()
	if err := service.identityService.LoadByID(authorization.IdentityID, &identity); err != nil {
		return false, derp.Wrap(err, location, "Error loading Identity for user")
	}

	// Find the products that are associated with the provided roles
	privileges := accessLister.RolesToProductIDs(roles...)

	// Return TRUE if the identity includes one or more of the required privileges
	return identity.HasPrivilege(privileges...), nil
}

// UserInGroup returns TRUE if the user is a member of the specified group
func (service *Permission) UserInGroup() (bool, error) {
	return false, derp.NotImplementedError("service.Permission.UserInGroup", "UserInGroup is not implemented")
}

// AuthorInGroup returns TRUE if the Author/AttributedTo is a member of the specified group
func (service *Permission) AuthorInGroup(string) (bool, error) {
	return false, derp.NotImplementedError("service.Permission.AuthorInGroup", "AuthorInGroup is not implemented")
}
