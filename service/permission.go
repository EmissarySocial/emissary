package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
)

type Permission struct {
	purchaseService *Purchase
}

func NewPermission() Permission {
	return Permission{}
}

func (service *Permission) Refresh(purchaseService *Purchase) {
	service.purchaseService = purchaseService
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

		// We can check for product ownership ONLY IF there is a valid GuestID
		guestID := authorization.GuestID

		if !guestID.IsZero() {

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

	spew.Dump(location)

	// Find the products that are associated with the provided roles
	productIDs := accessLister.RolesToProductIDs(roles...)

	spew.Dump("productIDs", productIDs)

	if productIDs.IsEmpty() {
		return false, nil // IF no products grant this role, then no access granted.
	}

	// Count how many of the eligible products the user has purchased
	count, err := service.purchaseService.CountByGuestAndProduct(authorization.GuestID, productIDs...)

	spew.Dump("count", count)
	spew.Dump("err", err)

	if err != nil {
		return false, derp.Wrap(err, location, "Error counting purchases for user")
	}

	spew.Dump("result", count > 0)

	// Return TRUE if the user has purchased on one or more eligible products
	return (count > 0), nil
}

// UserInGroup returns TRUE if the user is a member of the specified group
func (service *Permission) UserInGroup() (bool, error) {
	return false, derp.NotImplementedError("service.Permission.UserInGroup", "UserInGroup is not implemented")
}

// AuthorInGroup returns TRUE if the Author/AttributedTo is a member of the specified group
func (service *Permission) AuthorInGroup(string) (bool, error) {
	return false, derp.NotImplementedError("service.Permission.AuthorInGroup", "AuthorInGroup is not implemented")
}
