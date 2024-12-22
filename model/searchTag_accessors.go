package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTagSchema returns a validating schema for SearchTag objects
func SearchTagSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"searchTagId": schema.String{Format: "objectId"},
			"parent":      schema.String{},
			"name":        schema.String{Required: true},
			"description": schema.String{},
			"color":       schema.String{Format: "color"},
			"notes":       schema.String{},
			"stateId":     schema.Integer{Enum: []int{SearchTagStateFeatured, SearchTagStateAllowed, SearchTagStateWaiting, SearchTagStateBlocked}},
			"rank":        schema.Integer{},
		},
	}
}

// GetPointer implements the schema.PointerGetter interface,
// and allows read/write access to many fields of a SearchTag object
func (searchTag *SearchTag) GetPointer(name string) (any, bool) {

	switch name {

	case "name":
		return &searchTag.Name, true

	case "parent":
		return &searchTag.Parent, true

	case "stateId":
		return &searchTag.StateID, true

	case "description":
		return &searchTag.Description, true

	case "color":
		return &searchTag.Color, true

	case "notes":
		return &searchTag.Notes, true

	case "rank":
		return &searchTag.Rank, true
	}

	return nil, false
}

// GetStringOK implements the schema.StringGetter interface,
// and allows read access to many fields of a SearchTag object
func (searchTag SearchTag) GetStringOK(name string) (string, bool) {

	switch name {

	case "searchTagId":
		return searchTag.SearchTagID.Hex(), true
	}

	return "", false
}

// SetString implements the schema.StringSetter interface,
// and allows write access to many fields of a SearchTag object
func (searchTag *SearchTag) SetString(name string, value string) bool {

	switch name {

	case "searchTagId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			searchTag.SearchTagID = objectID
			return true
		}
	}

	return false
}
