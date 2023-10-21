package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func OAuthClientSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"clientId":      schema.String{Format: "objectId", Required: true},
			"applicationId": schema.String{Required: true},
			"name":          schema.String{Required: true},
			"website":       schema.String{Format: "url"},
			"scopes":        schema.Array{Items: schema.String{}},
		},
	}
}

func (app *OAuthClient) GetPointer(name string) (any, bool) {
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

func (app *OAuthClient) GetStringOK(name string) (string, bool) {

	switch name {

	case "clientId":
		return app.ClientID.Hex(), true
	}

	return "", false
}

func (app *OAuthClient) SetString(name string, value string) bool {

	switch name {

	case "clientId":
		if objectId, err := primitive.ObjectIDFromHex(value); err == nil {
			app.ClientID = objectId
			return true
		}
	}

	return false
}
