package service

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/hannibal/sigs"
)

// Permission service manages user permissions and privileges
type Permission struct {
	activityService  *ActivityStream
	identityService  *Identity
	privilegeService *Privilege
	userService      *User
}

// NewPermission returns a fully populated Permission service.
func NewPermission() Permission {
	return Permission{}
}

// Refresh updates links to additional services that may not have been initialized when this service was created.
func (service *Permission) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.identityService = factory.Identity()
	service.privilegeService = factory.Privilege()
	service.userService = factory.User()
}

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (service *Permission) UserCan(session data.Session, authorization *model.Authorization, template *model.Template, accessLister model.AccessLister, actionID string) (bool, error) {

	const location = "service.Permission.UserCan"

	// Find the action in the Template
	action, exists := template.Actions[actionID]

	if !exists {
		derp.Report(derp.Internal(location, "Action not found in template", "ActionID: "+actionID, "TemplateID: "+template.TemplateID))
		return false, nil
	}

	// Get a list of the valid roles for this action
	accessList := action.AccessList[accessLister.State()]

	// Map the list of allowed roles to a list of GroupIDs
	permissions := accessLister.RolesToGroupIDs(accessList...)

	if permissions.IsAnonymous() {
		return true, nil
	}

	// These checks are only valid if the request includes a UserID
	if authorization.IsAuthenticated() {

		// If the user is a domain owner, then they can do anything
		if authorization.DomainOwner {
			return true, nil
		}

		// If the permissions require an "authenticated" user, then allow the request
		if permissions.IsAuthenticated() {
			return true, nil
		}

		// Otherwise, check if the authorization includes one of the required permissions
		if authorization.IsGroupMember(permissions...) {
			return true, nil
		}
	}

	// These checks are only valid if the request includes an IdentityID
	if authorization.IsIdentity() {

		hasRole, err := service.hasPrivilege(session, authorization, accessLister, accessList...)

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

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (service *Permission) TraceUserCan(session data.Session, authorization *model.Authorization, template *model.Template, accessLister model.AccessLister, actionID string) []string {

	result := []string{"service.Permission.UserCan"}

	// Find the action in the Template
	action, exists := template.Actions[actionID]

	if !exists {
		result = append(result, "Action not found in template", "ActionID: "+actionID, "TemplateID: "+template.TemplateID)
		result = append(result, "FAILURE")
		return result
	}

	result = append(result, "action.AccessList:")
	for key, value := range action.AccessList {
		result = append(result, key+": "+value.Join(","))
	}

	// Get a list of the valid roles for this action
	accessList := action.AccessList[accessLister.State()]
	result = append(result, "AccessLister.State: "+accessLister.State())
	result = append(result, "AccessList:")
	result = append(result, accessList...)

	// Map the list of allowed roles to a list of GroupIDs
	permissions := accessLister.RolesToGroupIDs(accessList...)

	permissionsJSON, _ := json.Marshal(permissions)
	result = append(result, "Permissions: "+string(permissionsJSON))

	if permissions.IsAnonymous() {
		result = append(result, "Allow Anonymous", "SUCCESS")
	}

	// These checks are only valid if the request includes a UserID
	if authorization.IsAuthenticated() {

		result = append(result, "Authenticated User: "+authorization.UserID.Hex())

		// If the user is a domain owner, then they can do anything
		if authorization.DomainOwner {
			result = append(result, "Allow Domain Owner", "SUCCESS")
			return result
		}

		// If the permissions require an "authenticated" user, then allow the request
		if permissions.IsAuthenticated() {
			result = append(result, "Allow Authenticated", "SUCCESS")
			return result
		}

		// Otherwise, check if the authorization includes one of the required permissions
		if authorization.IsGroupMember(permissions...) {
			result = append(result, "Allow Group Member", "SUCCESS")
			return result
		}
	}

	// These checks are only valid if the request includes an IdentityID
	if authorization.IsIdentity() {

		result = append(result, "Identity: "+authorization.IdentityID.Hex())

		hasRole, err := service.hasPrivilege(session, authorization, accessLister, accessList...)

		if err != nil {
			result = append(result, "Error reading Privileges: "+err.Error())
			result = append(result, "FAILURE")
			return result
		}

		if hasRole {
			result = append(result, "Allow Identity with role", "SUCCESS")
			return result
		}
	}

	result = append(result, "User is not a Group Member or a Permitted Identity")
	return result
}

// UserHasRole returns TRUE if the user has access to the specified role
func (service *Permission) UserHasRole(session data.Session, authorization *model.Authorization, accessLister model.AccessLister, role string) (bool, error) {

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
		groupIDs := accessLister.RolesToGroupIDs(role) // nolint:scopeguard (readability)

		if authorization.IsGroupMember(groupIDs...) {
			return true, nil
		}
	}

	// If the authorization includes an IdentityID, then check that next
	if authorization.IsIdentity() {

		// See if this Identity has privileges for the specified role
		hasPrivilege, err := service.hasPrivilege(session, authorization, accessLister, role)

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
func (service *Permission) hasPrivilege(session data.Session, authorization *model.Authorization, accessLister model.AccessLister, requiredRoles ...string) (bool, error) {

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
	if err := service.identityService.LoadByID(session, authorization.IdentityID, &identity); err != nil {
		return false, derp.Wrap(err, location, "Unable to load Identity for user")
	}

	// Return TRUE if the identity includes one or more of the required privileges
	return identity.HasPrivilege(requiredPrivileges...), nil
}

// UserInGroup returns TRUE if the user is a member of the specified group
func (service *Permission) UserInGroup(authorization *model.Authorization, groupToken string) (bool, error) {
	return false, derp.NotImplemented("service.Permission.UserInGroup", "UserInGroup is not implemented")
}

// AuthorInGroup returns TRUE if the Author/AttributedTo is a member of the specified group
func (service *Permission) AuthorInGroup(accessLister model.AccessLister, groupToken string) (bool, error) {
	return false, derp.NotImplemented("service.Permission.AuthorInGroup", "AuthorInGroup is not implemented")
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

func (service *Permission) ParseHTTPSignature(session data.Session, request *http.Request) model.Permissions {

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
	if err := service.identityService.LoadByActivityPubActor(session, signature.ActorID(), &identity); err != nil {
		return result
	}

	// If present, then add the privileges for this Identity
	result = append(result, identity.PrivilegeIDs...)

	if !identity.HasEmailAddress() {
		return result
	}

	// If the Identity DOES have an email address, then look for a User, too
	user := model.NewUser()
	if err := service.userService.LoadByEmail(session, identity.EmailAddress, &user); err != nil {
		return result
	}

	result = append(result, model.MagicGroupIDAuthenticated, user.UserID)
	result = append(result, user.GroupIDs...)

	return result
}

func (service *Permission) getSignature(request *http.Request) (sigs.Signature, error) {

	const location = "service.Permission.getSignature"

	// First, try to verify the signature using the standard method
	signature, err := sigs.Verify(request, service.activityService.PublicKeyFinder)

	if err == nil {
		return signature, nil
	}

	// If there's an error on production servers, then fail
	if !dt.IsLocalhost(request.Host) {
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
