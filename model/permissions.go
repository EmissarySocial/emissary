package model

import (
	"github.com/benpate/rosetta/slice"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Permissions is a slice of ObjectIDs that represent the permissions granted to an OutboxMessage.
type Permissions []primitive.ObjectID

// NewPermissions returns fully initialized Permissions slice (with no permissions added)
func NewPermissions() Permissions {
	return make(Permissions, 0)
}

// NewPermissions returns a fully initialized Permissions slice with "anonymous" permissions included.
func NewAnonymousPermissions() Permissions {
	return Permissions{MagicGroupIDAnonymous}
}

func NewAuthenticatedPermissions() Permissions {
	return Permissions{MagicGroupIDAuthenticated}
}

// IsZero returns TRUE if this Permissions slice is empty.
func (permissions Permissions) IsZero() bool {
	return len(permissions) == 0
}

// IsAnonymous returns TRUE if this Permisssions slice allows "anonymous" access.
func (permissions Permissions) IsAnonymous() bool {
	return slice.Contains(permissions, MagicGroupIDAnonymous)
}

// IsAuthenticated returns TRUE if this Permissions slice allows "authenticated" access.
func (permissions Permissions) IsAuthenticated() bool {
	return slice.Contains(permissions, MagicGroupIDAuthenticated)
}
