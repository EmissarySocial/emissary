package model

import (
	"github.com/benpate/datatype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Criteria struct {
	Groups  map[string][]string `path:"groups"  json:"groups"  bson:"groups"`  // A map of groupIDs to the roles that each group can use
	OwnerID primitive.ObjectID  `path:"ownerId" json:"ownerId" bson:"ownerId"` // UserID of the person who owns this content.
}

func NewCriteria() Criteria {
	return Criteria{
		Groups: make(map[string][]string),
	}
}

// FindGroups returns all groups that match the provided roles
func (criteria *Criteria) FindGroups(roles ...string) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0)

	for groupID, groupRoles := range criteria.Groups {
		if matchAny(roles, groupRoles) {
			if groupID, err := primitive.ObjectIDFromHex(groupID); err == nil {
				result = append(result, groupID)
			}
		}
	}

	return result
}

// Roles returns a unique list of all roles that the provided groups can access.
func (criteria *Criteria) Roles(groupIDs ...primitive.ObjectID) []string {

	result := []string{}

	// Copy values from group roles
	for _, groupID := range groupIDs {
		if roles, ok := criteria.Groups[groupID.Hex()]; ok {
			result = append(result, roles...)
		}
	}

	return result
}

// SimpleModel returns a model object for displaying Simple Sharing.
func (criteria *Criteria) SimpleModel() datatype.Map {

	// Special case if this is for EVERYBODY
	if _, ok := criteria.Groups[MagicGroupIDAnonymous.Hex()]; ok {
		return datatype.Map{
			"rule":     "anonymous",
			"groupIds": []string{},
		}
	}

	// Special case if this is for AUTHENTICATED
	if _, ok := criteria.Groups[MagicGroupIDAuthenticated.Hex()]; ok {
		return datatype.Map{
			"rule":     "authenticated",
			"groupIds": []string{},
		}
	}

	// Fall through means that additional groups are selected.
	// First, get all keys to the Groups map
	groupIDs := make([]string, len(criteria.Groups))
	index := 0

	for groupID := range criteria.Groups {
		groupIDs[index] = groupID
		index++
	}

	return datatype.Map{
		"rule":     "private",
		"groupIds": groupIDs,
	}
}
