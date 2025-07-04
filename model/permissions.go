package model

import (
	"github.com/benpate/rosetta/schema"
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

// IsZero returns TRUE if this Permissions slice has zero items.
func (permissions Permissions) IsZero() bool {
	return len(permissions) == 0
}

// NotZero returns TRUE if this Permissions slice has at least one item.
func (permissions Permissions) NotZero() bool {
	return len(permissions) > 0
}

func (permissions Permissions) Length() int {
	return len(permissions)
}

func (permissions Permissions) IsLength(length int) bool {
	return len(permissions) == length
}

// IsAnonymous returns TRUE if this Permisssions slice allows "anonymous" access.
func (permissions Permissions) IsAnonymous() bool {
	return slice.Contains(permissions, MagicGroupIDAnonymous)
}

// IsAuthenticated returns TRUE if this Permissions slice allows "authenticated" access.
func (permissions Permissions) IsAuthenticated() bool {
	return slice.Contains(permissions, MagicGroupIDAuthenticated)
}

func (permissions Permissions) Intersects(other Permissions) bool {
	return slice.ContainsAny(permissions, other...)
}

func (permissions Permissions) First() primitive.ObjectID {
	if permissions.IsZero() {
		return primitive.NilObjectID
	}

	return permissions[0]
}

func (permissions Permissions) GetStringOK(name string) (string, bool) {

	if index, ok := schema.Index(name, permissions.Length()); ok {
		return permissions[index].Hex(), true
	}

	return "", false
}

func (permissions *Permissions) SetString(name string, value string) bool {

	if objectID, err := primitive.ObjectIDFromHex(value); err == nil {

		if index, ok := schema.Index(name); ok {

			for index >= permissions.Length() {
				(*permissions) = append(*permissions, primitive.NilObjectID)
			}

			(*permissions)[index] = objectID
			return true
		}
	}

	return false
}
