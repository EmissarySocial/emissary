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
	accessList := action.AccessList[accessLister.State()]

	// If Anonymous access is allowed, then EVERYONE can perform this action
	if accessList.Anonymous {
		return true, nil
	}

	//////////////////////////////////////////////////////////////////
	// Beyond this point, you must be logged in to perform this action

	// These checks are only valid if the page request is authenticated (via a UserID)
	// If the user is a domain owner, then they can do anything
	if authorization.DomainOwner {
		return true, nil
	}

	// If the accessList allows "authenticated" access, then any authenticated user can perform this action
	if accessList.Authenticated {
		if authorization.IsAuthenticated() {
			return true, nil
		}
	}

	// If the accessList allows "author" access, then check to see if the user is the author of this object
	if accessList.Author {
		if accessLister.IsAuthor(authorization.UserID) {
			return true, nil
		}
	}

	// If the accessList allows "myself" access, then check to see if this user is "myself"
	if accessList.Self {
		if accessLister.IsMyself(authorization.UserID) {
			return true, nil
		}
	}

	// Check for group access via the Authorization object
	if len(accessList.Groups) > 0 {

		// Map the list of allowed roles to a list of GroupIDs
		groupIDs := accessLister.RolesToGroupIDs(accessList.Groups...)

		if authorization.IsGroupMember(groupIDs...) {
			return true, nil
		}
	}

	// These checks are only valid if the page request includes an Identity (via an IdentityID)
	if authorization.IsIdentity() {

		hasRole, err := service.hasPrivilege(authorization, accessLister, accessList.Privileges...)

		if err != nil {
			return false, derp.Wrap(err, location, "Unable to check user roles")
		}

		if hasRole {
			return true, nil
		}
	}

	// The user does not have permission to perform this action
	return false, nil
}

// UserHasRole returns TRUE if the user has access to the specified role
func (service *Permission) UserHasRole(authorization *model.Authorization, accessLister model.AccessLister, role string) (bool, error) {

	const location = "service.Permission.UserHasRole"

	switch role {

	case model.MagicRoleAnonymous:
		return true, nil

	case model.MagicRoleAuthenticated:
		return authorization.IsAuthenticated(), nil

	case model.MagicRoleAuthor:
		return accessLister.IsAuthor(authorization.UserID), nil

	case model.MagicRoleMyself:
		return accessLister.IsMyself(authorization.UserID), nil

	case model.MagicRoleOwner:
		return authorization.DomainOwner, nil
	}

	// If the authorization includes GroupIDs, then check those next
	if authorization.GroupIDs.NotEmpty() {

		// See if any of these roles are associated with the Groups from the Authorization
		groupIDs := accessLister.RolesToGroupIDs(role)

		if authorization.IsGroupMember(groupIDs...) {
			return true, nil
		}
	}

	// If the authorization includes an IdentityID, then check that next
	if authorization.IsIdentity() {

		// See if this Identity has privileges for the specified role
		hasPrivilege, err := service.hasPrivilege(authorization, accessLister, role)

		if err != nil {
			return false, derp.Wrap(err, location, "Unable to check user roles")
		}

		if hasPrivilege {
			return true, nil
		}
	}

	return false, nil
}

// HasPrivilege returns TRUE if the user has privileges for the specified role
func (service *Permission) hasPrivilege(authorization *model.Authorization, accessLister model.AccessLister, requiredRoles ...string) (bool, error) {

	const location = "service.Permission.HasPrivilege"

	// If no roles are provided then the user does not have permission
	if len(requiredRoles) == 0 {
		return false, nil
	}

	// Locate the authorized Identity
	identity := model.NewIdentity()
	if err := service.identityService.LoadByID(authorization.IdentityID, &identity); err != nil {
		return false, derp.Wrap(err, location, "Error loading Identity for user")
	}

	// Find the products that are associated with the provided roles
	requiredPrivileges := accessLister.RolesToPrivileges(requiredRoles...)

	// Return TRUE if the identity includes one or more of the required privileges
	return identity.HasPrivilege(requiredPrivileges...), nil
}

// UserInGroup returns TRUE if the user is a member of the specified group
func (service *Permission) UserInGroup(authorization *model.Authorization, groupToken string) (bool, error) {
	return false, derp.NotImplementedError("service.Permission.UserInGroup", "UserInGroup is not implemented")
}

// AuthorInGroup returns TRUE if the Author/AttributedTo is a member of the specified group
func (service *Permission) AuthorInGroup(accessLister model.AccessLister, groupToken string) (bool, error) {
	return false, derp.NotImplementedError("service.Permission.AuthorInGroup", "AuthorInGroup is not implemented")
}
