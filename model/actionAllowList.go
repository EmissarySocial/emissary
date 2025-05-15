package model

import "github.com/benpate/rosetta/sliceof"

// ActionAllowList is a white-list for the ROLES that are allowed to perform a given ACTION.
// IMPORTANT: By default, no roles are allowed to perform any actions, so if no allow list is
// provided then it will only be accessible to Domain Owners.
type ActionAllowList struct {
	Anonymous     bool
	Authenticated bool
	Author        bool
	Self          bool
	GroupRoles    sliceof.String
	ProductRoles  sliceof.String
}

// NewActionAllowList returns a fully initialized ActionAllowList
func NewActionAllowList() ActionAllowList {
	return ActionAllowList{
		GroupRoles:   make(sliceof.String, 0),
		ProductRoles: make(sliceof.String, 0),
	}
}

// Roles returns a list of all Roles that are allowed by this AllowList.
func (allowList ActionAllowList) Roles() sliceof.String {

	result := make(sliceof.String, 0)

	if allowList.Anonymous {
		result = append(result, MagicRoleAnonymous)
	}

	if allowList.Authenticated {
		result = append(result, MagicRoleAuthenticated)
	}

	if allowList.Author {
		result = append(result, MagicRoleAuthor)
	}

	if allowList.Self {
		result = append(result, MagicRoleMyself)
	}

	if len(allowList.GroupRoles) > 0 {
		result = append(result, allowList.GroupRoles...)
	}

	if len(allowList.ProductRoles) > 0 {
		result = append(result, allowList.ProductRoles...)
	}

	return result
}
