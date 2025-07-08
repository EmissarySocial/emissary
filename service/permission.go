package service

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/sigs"
)

type Permission struct {
	activityStream   *ActivityStream
	identityService  *Identity
	privilegeService *Privilege
	userService      *User
}

func NewPermission() Permission {
	return Permission{}
}

func (service *Permission) Refresh(activityStream *ActivityStream, identityService *Identity, privilegeService *Privilege, userService *User) {
	service.activityStream = activityStream
	service.identityService = identityService
	service.privilegeService = privilegeService
	service.userService = userService
}

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (service *Permission) UserCan(authorization *model.Authorization, template *model.Template, accessLister model.AccessLister, actionID string) (bool, error) {

	const location = "service.Permission.UserCan"

	// Find the action in the Template
	action, exists := template.Actions[actionID]

	if !exists {
		derp.Report(derp.InternalError(location, "Action not found in template", "ActionID: "+actionID, "TemplateID: "+template.TemplateID))
		return false, nil
	}

	// Get a list of the valid roles for this action
	accessList := action.AccessList[accessLister.State()]

	// If Anonymous access is allowed, then EVERYONE can perform this action
	if accessList.Anonymous {
		return true, nil
	}

	// These checks are only valid if the request includes a UserID
	if authorization.IsAuthenticated() {

		// These checks are only valid if the page request is authenticated (via a UserID)
		// If the user is a domain owner, then they can do anything
		if authorization.DomainOwner {
			return true, nil
		}

		// If the accessList allows "authenticated" access, then any authenticated user can perform this action
		if accessList.Authenticated {
			return true, nil
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
	}

	// These checks are only valid if the request includes an IdentityID
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

	// Find the products that are associated with the provided roles
	requiredPrivileges := accessLister.RolesToPrivilegeIDs(requiredRoles...)

	if requiredPrivileges.IsZero() {
		return false, nil // No privileges associated with this role
	}

	// Locate the authorized Identity
	identity := model.NewIdentity()
	if err := service.identityService.LoadByID(authorization.IdentityID, &identity); err != nil {
		return false, derp.Wrap(err, location, "Error loading Identity for user")
	}

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

func (service *Permission) Permissions(authorization *model.Authorization, identity *model.Identity) model.Permissions {

	result := model.NewAnonymousPermissions()

	if authorization != nil {

		// Domain owners can see every valid object. Do not touch the criteria
		if authorization.DomainOwner {
			return model.NewPermissions()
		}

		if authorization.IsAuthenticated() {
			result = append(result, model.MagicGroupIDAuthenticated, authorization.UserID)
			result = append(result, authorization.GroupIDs...)
		}
	}

	// If an identity is provided, then include all of the Privileges for this Identity
	if identity != nil {
		result = append(result, identity.PrivilegeIDs...)
	}

	return result
}

func (service *Permission) ParseHTTPSignature(request *http.Request) model.Permissions {

	result := model.NewAnonymousPermissions()

	// RULE: Empty requests are not signed.  This should never happen..
	if request == nil {
		return result
	}

	// Verify the signature
	signature, err := service.getSignature(request)

	if err != nil {
		return result
	}

	// Find an Identity based on the signature
	identity := model.NewIdentity()
	if err := service.identityService.LoadByActivityPubActor(signature.ActorID(), &identity); err != nil {
		return result
	}

	// If present, then add the privileges for this Identity
	result = append(result, identity.PrivilegeIDs...)

	if !identity.HasEmailAddress() {
		return result
	}

	// If the Identity DOES have an email address, then look for a User, too
	user := model.NewUser()
	if err := service.userService.LoadByEmail(identity.EmailAddress, &user); err != nil {
		return result
	}

	result = append(result, model.MagicGroupIDAuthenticated, user.UserID)
	result = append(result, user.GroupIDs...)

	return result
}

func (service *Permission) getSignature(request *http.Request) (sigs.Signature, error) {

	const location = "service.Permission.getSignature"

	// First, try to verify the signature using the standard method
	verifier := sigs.NewVerifier()

	signature, err := verifier.Verify(request, service.activityStream.PublicKeyFinder)

	if err == nil {
		return signature, nil
	}

	// If there's an error on production servers, then fail
	if !domain.IsLocalhost(request.Host) {
		return sigs.Signature{}, derp.Wrap(err, location, "Unable to verify signature for request")
	}

	// Fall through means we're on localhost; try the "mock" verifier
	if mockKeyID := request.Header.Get("Mock-Key-Id"); mockKeyID != "" {
		result := sigs.Signature{
			KeyID:     mockKeyID,
			Algorithm: "MOCK",
			Headers:   make([]string, 0),
			Signature: make([]byte, 0),
			Expires:   math.MaxInt64,
		}

		return result, nil
	}

	return sigs.Signature{}, derp.Wrap(err, location, "No valid signature found. For local domains, use 'Mock-Key-Id' header to simulate a signing key.")
}
