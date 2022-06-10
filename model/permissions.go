package model

import (
	"github.com/benpate/datatype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Permissions map[string][]string

func NewPermissions() Permissions {
	return make(Permissions)
}

func (criteria Permissions) Assign(role string, groupID primitive.ObjectID) {
	groupIDHex := groupID.Hex()

	if _, ok := criteria[groupIDHex]; !ok {
		criteria[groupIDHex] = []string{role}
		return
	}

	criteria[groupIDHex] = append(criteria[groupIDHex], role)
}

// Groups returns all groups that match the provided roles
func (criteria Permissions) Groups(roles ...string) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0)

	for _, role := range roles {
		switch role {
		case "anonymous":
			result = append(result, MagicGroupIDAnonymous)
		case "authenticated":
			result = append(result, MagicGroupIDAuthenticated)
		}
	}

	for groupID, groupRoles := range criteria {
		if matchAny(roles, groupRoles) {
			if groupID, err := primitive.ObjectIDFromHex(groupID); err == nil {
				result = append(result, groupID)
			}
		}
	}

	return result
}

// Roles returns a unique list of all roles that the provided groups can access.
func (criteria Permissions) Roles(groupIDs ...primitive.ObjectID) []string {

	result := []string{}

	// Copy values from group roles
	for _, groupID := range groupIDs {
		if roles, ok := criteria[groupID.Hex()]; ok {
			result = append(result, roles...)
		}
	}

	return result
}

// SimpleModel returns a model object for displaying Simple Sharing.
func (criteria Permissions) SimpleModel() datatype.Map {

	// Special case if this is for EVERYBODY
	if _, ok := criteria[MagicGroupIDAnonymous.Hex()]; ok {
		return datatype.Map{
			"rule":     "anonymous",
			"groupIds": []string{},
		}
	}

	// Special case if this is for AUTHENTICATED
	if _, ok := criteria[MagicGroupIDAuthenticated.Hex()]; ok {
		return datatype.Map{
			"rule":     "authenticated",
			"groupIds": []string{},
		}
	}

	// Fall through means that additional groups are selected.
	// First, get all keys to the Groups map
	groupIDs := make([]string, len(criteria))
	index := 0

	for groupID := range criteria {
		groupIDs[index] = groupID
		index++
	}

	return datatype.Map{
		"rule":     "private",
		"groupIds": groupIDs,
	}
}
