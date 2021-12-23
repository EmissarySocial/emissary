package model

import (
	"github.com/benpate/compare"
	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Criteria struct {
	Inherit bool                `json:"inherit"  bson:"inherit"` // If TRUE, then this criteria is inherited from a parent stream
	Public  bool                `json:"public"   bson:"public"`  // If TRUE, then no permissions are required to view this stream
	Groups  map[string][]string `json:"roles"    bson:"roles"`   // A map of groupIDs -> the roles that each group can access
}

func NewCriteria() Criteria {
	return Criteria{
		Inherit: true,
		Public:  false,
		Groups:  make(map[string][]string),
	}
}

// Roles returns a unique list of all roles that the provided groups can access.
func (criteria *Criteria) Roles(groupIDs ...primitive.ObjectID) []string {

	result := make([]string, 0)

	for _, groupID := range groupIDs {
		if roles, ok := criteria.Groups[groupID.Hex()]; ok {
			for _, role := range roles {
				if !compare.Contains(result, role) {
					result = append(result, role)
				}
			}
		}
	}

	return result
}

func (criteria *Criteria) SimpleModel() datatype.Map {

	groupIDs := make([]string, len(criteria.Groups))
	index := 0

	for groupID := range criteria.Groups {
		groupIDs[index] = groupID
		index++
	}

	return datatype.Map{
		"public":   criteria.Public,
		"groupIds": groupIDs,
	}
}

func (criteria *Criteria) GetPath(p path.Path) (interface{}, error) {

	switch p.Head() {

	case "inherit":
		return criteria.Inherit, nil

	case "public":
		return criteria.Public, nil

		/*
			case "roles":

				role, p := p.Split()

				if !p.HasTail() {
					return criteria.Roles[role], nil
				}

				indexInterface, p := p.Split()
				index, ok := convert.IntOk(indexInterface, 0)

				if !ok {
					return nil, derp.New(500, "ghost.model.Criteria.GetPath", "Invalid index", p)
				}

				if index >= len(criteria.Roles[role]) {
					return nil, derp.New(500, "ghost.model.Criteria.GetPath", "Invalid index", p)
				}

				return criteria.Roles[role][index], nil
		*/
	}

	return nil, derp.New(500, "ghost.model.Criteria.GetPath", "Unrecognied Path", p)
}

func (criteria *Criteria) SetPath(p path.Path, value interface{}) error {

	switch p.Head() {

	case "inherit":
		criteria.Inherit = convert.Bool(value)

	case "public":
		criteria.Public = convert.Bool(value)

		/*
			case "roles":

				if !p.HasTail() {
					valueMap, ok := value.(map[string]interface{})

					if !ok {
						return derp.New(500, "ghost.model.Criteria.SetPath", "Invalid data format for roles", p, value)
					}

					for key, item := range valueMap {
						objectIDs, err := datatype.ParseObjectIDList(item)

						if err != nil {
							return derp.Wrap(err, "ghost.model.Criteria.SetPath", "Invalid data format for role")
						}

						criteria.Roles[key] = objectIDs
					}
					return nil
				}

				role, p := p.Split()

				if !p.HasTail() {

					objectIDs, err := datatype.ParseObjectIDList(value)

					if err != nil {
						return derp.Wrap(err, "ghost.model.Criteria.SetPath", "Invalid data format for role")
					}

					criteria.Roles[role] = objectIDs
					return nil
				}

				return criteria.Roles[role].SetPath(p, value)
		*/
	}

	return derp.New(500, "ghost.model.Criteria.SetPath", "Unrecognied Path", p)
}
