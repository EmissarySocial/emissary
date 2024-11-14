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
			"parentId":    schema.String{Format: "objectId"},
			"tag":         schema.String{Required: true},
			"notes":       schema.String{},
			"stateId":     schema.Integer{Required: true, Enum: []int{SearchTagStateFeatured, SearchTagStateAllowed, SearchTagStateWaiting, SearchTagStateBlocked}},
			"rank":        schema.Integer{},
		},
	}
}

// GetPointer implements the schema.PointerGetter interface,
// and allows read/write access to many fields of a SearchTag object
func (searchTag *SearchTag) GetPointer(name string) (any, bool) {

	switch name {

	case "tag":
		return &searchTag.Tag, true

	case "stateId":
		return &searchTag.StateID, true

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

	case "parentId":
		return searchTag.ParentID.Hex(), true

	case "tag":
		return searchTag.Tag, true

	case "notes":
		return searchTag.Notes, true
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

	case "parentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			searchTag.ParentID = objectID
			return true
		}

	case "tag":
		searchTag.Tag = value
		return true

	case "notes":
		searchTag.Notes = value
		return true
	}

	return false
}
