package model

import (
	"github.com/benpate/rosetta/sliceof"
)

// AccessList is a white-list for the ROLES that are allowed to perform a given ACTION.
// IMPORTANT: By default, no roles are allowed to perform any actions, so if no allow list is
// provided then it will only be accessible to Domain Owners.
type AccessList struct {
	Groups        sliceof.String
	Privileges    sliceof.String
	Anonymous     bool
	Authenticated bool
	Author        bool
	Self          bool
}

// NewAccessList returns a fully initialized AccessList
func NewAccessList() AccessList {
	return AccessList{
		Groups:     sliceof.NewString(),
		Privileges: sliceof.NewString(),
	}
}

// Roles returns a slice containing all of the roles that are valid for this AccessList.
func (accessList AccessList) Roles() sliceof.String {

	if accessList.Anonymous {
		return sliceof.String{MagicRoleAnonymous}
	}

	if accessList.Authenticated {
		return sliceof.String{MagicRoleAuthenticated}
	}

	result := sliceof.NewString()

	if accessList.Author {
		result = append(result, MagicRoleAuthor)
	}

	if accessList.Self {
		result = append(result, MagicRoleMyself)
	}

	result = append(result, accessList.Groups...)
	result = append(result, accessList.Privileges...)
	return result
}
