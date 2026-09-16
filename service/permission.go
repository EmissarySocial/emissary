package service

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/uri"
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
			return false, derp.Wrap(err, location, "Checking user roles")
		}

		if hasRole {
			return true, nil
		}
	}

	// The user does not have permission to perform this action
	return false, nil
}

// TraceUserCan returns the step-by-step reasoning behind a UserCan decision, for debugging.
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
			return false, derp.Wrap(err, location, "Checking user roles")
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
		return false, derp.Wrap(err, location, "Loading Identity for user")
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

// Permissions returns the full set of Permissions granted by an Authorization and an Identity.
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

// ParseHTTPSignature returns the Permissions granted by a request's HTTP Signature, refusing
// any request that carries a signature it cannot verify.
func (service *Permission) ParseHTTPSignature(session data.Session, request *http.Request) (model.Permissions, error) {

	result := model.NewAnonymousPermissions()

	// RULE: Empty requests are not signed.  This should never happen..
	if request == nil {
		return result, nil
	}

	// Verify the signature
	signature, err := service.getSignature(request)

	// RULE: A signature that is present but INVALID refuses the whole request. The refusal
	// travels back unwrapped, so the peer reads its fixed message instead of a wrapper's. (BUG-20)
	if err != nil {
		return result, err
	}

	// RULE: A request that offers no Signature at all is Anonymous, and names no Actor to load
	if signature.KeyID == "" {
		return result, nil
	}

	// Find an Identity based on the signature. A signature that names no local Identity is
	// still a valid signature; it simply carries no extra privileges here.
	identity := model.NewIdentity()
	if err := service.identityService.LoadByActivityPubActor(session, signature.ActorID(), &identity); err != nil {
		return result, nil
	}

	// If present, then add the privileges for this Identity
	result = append(result, identity.PrivilegeIDs...)

	if !identity.HasEmailAddress() {
		return result, nil
	}

	// If the Identity DOES have an email address, then look for a User, too
	user := model.NewUser()
	if err := service.userService.LoadByEmail(session, identity.EmailAddress, &user); err != nil {
		return result, nil
	}

	result = append(result, model.MagicGroupIDAuthenticated, user.UserID)
	result = append(result, user.GroupIDs...)

	return result, nil
}

// getSignature verifies and returns the HTTP signature on an inbound request, or an empty
// Signature when the request carries no signature at all.
func (service *Permission) getSignature(request *http.Request) (sigs.Signature, error) {
	return resolveSignature(request, service.activityService.VerifySignature)
}

//////////////////////////////////////////
// Helper functions
//////////////////////////////////////////

// resolveSignature separates the three cases an inbound signature can present: no signature
// (Anonymous), a valid signature (an Actor), and one that FAILS to verify (a refusal).
// separated for testability.
func resolveSignature(request *http.Request, verify func(*http.Request) (sigs.Signature, error)) (sigs.Signature, error) {

	const location = "service.resolveSignature"

	isSigned := sigs.HasSignature(request)

	// A signature that verifies speaks for its Actor. A verification FAILURE is not returned
	// here, because the mock verifier below may still stand in for it on a local domain.
	if isSigned {
		if signature, err := verify(request); err == nil {
			return signature, nil
		}
	}

	// RULE: A local domain accepts a mock key in place of a real signature. This branch is
	// reached by UNSIGNED requests too, so it must sit ahead of the rules below -- a local
	// harness names its actor with this header alone. (BUG-51)
	if uri.IsLocalHostname(request.Host) {
		if mockKeyID := request.Header.Get("Mock-Key-Id"); mockKeyID != "" {
			return mockSignature(mockKeyID), nil
		}
	}

	// RULE: A request that offers no Signature at all is Anonymous, not refused
	if !isSigned {
		return sigs.Signature{}, nil
	}

	// RULE: A local domain gets the mock-key hint, which is the only guidance a developer
	// receives here -- errorHandler answers a 401 with this message and nothing else
	if uri.IsLocalHostname(request.Host) {
		return sigs.Signature{}, derp.Unauthorized(location, "Invalid HTTP Signature. For local domains, use the 'Mock-Key-Id' header to simulate a signing key")
	}

	// RULE: A signature that is present but INVALID refuses the whole request. Falling through
	// to anonymous hands the peer a normal-looking 200, or a 403 naming the wrong cause, with no
	// hint that their signature was rejected. Nothing is logged or reported: the 401 IS the
	// signal, and its message is fixed so verifier internals never reach a prober. (BUG-20)
	return sigs.Signature{}, derp.Unauthorized(location, "Invalid HTTP Signature")
}

// mockSignature returns the stand-in Signature that a local domain accepts in place of a real one.
func mockSignature(keyID string) sigs.Signature {
	return sigs.Signature{
		KeyID:     keyID,
		Algorithm: "MOCK",
		Headers:   make([]string, 0),
		Signature: make([]byte, 0),
		Expires:   math.MaxInt64,
	}
}
