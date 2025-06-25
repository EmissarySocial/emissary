package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CircleSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"circleId":    schema.String{Format: "objectId", Required: true},
			"userId":      schema.String{Format: "objectId", Required: true},
			"name":        schema.String{MaxLength: 64, Required: true},
			"color":       schema.String{MaxLength: 32, Default: "#000000"},
			"icon":        schema.String{MaxLength: 64, Default: "circle", Required: true},
			"description": schema.String{MaxLength: 2048},
			"productIds":  schema.Array{Items: schema.String{}},
			"isVisible":   schema.Boolean{},
			"isFeatured":  schema.Boolean{},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (circle *Circle) GetStringOK(name string) (string, bool) {

	switch name {

	case "circleId":
		return circle.CircleID.Hex(), true

	case "userId":
		return circle.UserID.Hex(), true

	case "name":
		return circle.Name, true

	case "color":
		return circle.Color, true

	case "icon":
		return circle.Icon, true

	case "description":
		return circle.Description, true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (circle *Circle) SetString(name string, value string) bool {

	switch name {

	case "circleId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			circle.CircleID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			circle.UserID = objectID
			return true
		}

	case "name":
		circle.Name = value
		return true

	case "color":
		circle.Color = value
		return true

	case "icon":
		circle.Icon = value
		return true

	case "description":
		circle.Description = value
		return true

	}

	return false
}

func (circle *Circle) GetPointer(name string) (any, bool) {

	switch name {

	case "name":
		return &circle.Name, true

	case "color":
		return &circle.Color, true

	case "icon":
		return &circle.Icon, true

	case "description":
		return &circle.Description, true

	case "productIds":
		return &circle.ProductIDs, true

	case "isVisible":
		return &circle.IsVisible, true

	case "isFeatured":
		return &circle.IsFeatured, true
	}

	return nil, false
}
