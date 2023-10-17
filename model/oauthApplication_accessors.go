package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func OAuthApplicationSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"oauthApplicationID": schema.String{Format: "objectId", Required: true},
			"applicationID":      schema.String{Required: true},
			"name":               schema.String{Required: true},
			"website":            schema.String{Format: "url"},
			"scopes":             schema.Array{Items: schema.String{}},
		},
	}
}

func (app *OAuthApplication) GetPointer(name string) (any, bool) {
	switch name {

	case "name":
		return &app.Name, true

	case "website":
		return &app.Website, true

	case "scopes":
		return &app.Scopes, true
	}

	return nil, false
}

func (app *OAuthApplication) GetStringOK(name string) (string, bool) {

	switch name {

	case "oauthApplicationId":
		return app.OAuthApplicationID.Hex(), true
	}

	return "", false
}

func (app *OAuthApplication) SetString(name string, value string) bool {

	switch name {

	case "oauthApplicationId":
		if objectId, err := primitive.ObjectIDFromHex(value); err == nil {
			app.OAuthApplicationID = objectId
			return true
		}
	}

	return false
}
