package model

import "github.com/benpate/rosetta/sliceof"

type ActionAllowList struct {
	Anonymous     bool
	Authenticated bool
	Author        bool
	Self          bool
	GroupRoles    sliceof.String
	ProductRoles  sliceof.String
}

func NewActionAllowList() ActionAllowList {
	return ActionAllowList{
		GroupRoles:   make(sliceof.String, 0),
		ProductRoles: make(sliceof.String, 0),
	}
}

// IsZero returns TRUE if this ActionAllowList is empty
func (allowList ActionAllowList) IsZero() bool {

	if allowList.Anonymous {
		return false
	}

	if allowList.Authenticated {
		return false
	}

	if allowList.Author {
		return false
	}

	if allowList.Self {
		return false
	}

	if len(allowList.GroupRoles) > 0 {
		return false
	}

	if len(allowList.ProductRoles) > 0 {
		return false
	}

	return true
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
