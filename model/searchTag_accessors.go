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
			"group":       schema.String{MaxLength: 64},
			"name":        schema.String{MaxLength: 64, Required: true},
			"stateId":     schema.Integer{Enum: []int{SearchTagStateFeatured, SearchTagStateAllowed, SearchTagStateWaiting, SearchTagStateBlocked}},
			"related":     schema.String{MaxLength: 256},
			"rank":        schema.Integer{},
			"colors":      schema.Array{Items: schema.String{Format: "color"}},
			"notes":       schema.String{MaxLength: 2048},
			"imageId":     schema.String{MaxLength: 1024, Format: "objectId"},
			"imageUrl":    schema.String{MaxLength: 1024}, // virtual field; ImageURL() returns a relative path, so the absolute-only "url" format cannot apply
		},
	}
}

// GetPointer implements the schema.PointerGetter interface,
// and allows read/write access to many fields of a SearchTag object
func (searchTag *SearchTag) GetPointer(name string) (any, bool) {

	switch name {

	case "group":
		return &searchTag.Group, true

	case "name":
		return &searchTag.Name, true

	case "stateId":
		return &searchTag.StateID, true

	case "related":
		return &searchTag.Related, true

	case "rank":
		return &searchTag.Rank, true

	case "colors":
		return &searchTag.Colors, true

	case "notes":
		return &searchTag.Notes, true

	}

	return nil, false
}

// GetStringOK implements the schema.StringGetter interface,
// and allows read access to many fields of a SearchTag object
func (searchTag SearchTag) GetStringOK(name string) (string, bool) {

	switch name {

	case "searchTagId":
		return searchTag.SearchTagID.Hex(), true

	case "imageId":
		return searchTag.ImageID.Hex(), true

	case "imageUrl":
		return searchTag.ImageURL(), true
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

	case "imageId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			searchTag.ImageID = objectID
			return true
		}

	// Fail silently when "setting" this virtual field
	case "imageUrl":
		return true
	}

	return false
}
