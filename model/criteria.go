package model

import (
	"github.com/benpate/datatype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Criteria struct {
	Public  []string            `path:"public"  json:"public"  bson:"public"`  // An array of roles that PUBLIC users can use
	Groups  map[string][]string `path:"groups"  json:"roles"   bson:"roles"`   // A map of groupIDs to the roles that each group can use
	OwnerID primitive.ObjectID  `path:"ownerId" json:"ownerId" bson:"ownerId"` // UserID of the person who owns this content.
}

func NewCriteria() Criteria {
	return Criteria{
		Public: make([]string, 0),
		Groups: make(map[string][]string),
	}
}

// Roles returns a unique list of all roles that the provided groups can access.
func (criteria *Criteria) Roles(groupIDs ...primitive.ObjectID) []string {

	result := make([]string, len(criteria.Public))

	// Copy values from public roles
	for index := range criteria.Public {
		result[index] = criteria.Public[index]
	}

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

	// If there are PUBLIC roles, then public box is checked.
	if len(criteria.Public) > 0 {
		return datatype.Map{
			"public":   true,
			"groupIds": []string{},
		}
	}

	// Fall through means that additional groups are selected.
	groupIDs := make([]string, len(criteria.Groups))
	index := 0

	for groupID := range criteria.Groups {
		groupIDs[index] = groupID
		index++
	}

	return datatype.Map{
		"public":   false,
		"groupIds": groupIDs,
	}
}
