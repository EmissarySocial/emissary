package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func OAuthClientSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"clientId":     schema.String{Format: "objectId", Required: true},
			"clientSecret": schema.String{},
			"actorId":      schema.String{},
			"name":         schema.String{Required: true},
			"summary":      schema.String{},
			"iconUrl":      schema.String{Format: "url"},
			"website":      schema.String{Format: "url"},
			"redirectUris": schema.Array{Items: schema.String{Format: "url"}},
			"scopes":       schema.Array{Items: schema.String{}},
		},
	}
}

func (client *OAuthClient) GetPointer(name string) (any, bool) {
	switch name {

	case "actorId":
		return &client.ActorID, true

	case "name":
		return &client.Name, true

	case "summary":
		return &client.Summary, true

	case "iconUrl":
		return &client.IconURL, true

	case "website":
		return &client.Website, true

	case "redirectUris":
		return &client.RedirectURIs, true

	case "scopes":
		return &client.Scopes, true
	}

	return nil, false
}

func (client *OAuthClient) GetStringOK(name string) (string, bool) {

	switch name {

	case "clientId":
		return client.ClientID.Hex(), true
	}

	return "", false
}

func (client *OAuthClient) SetString(name string, value string) bool {

	switch name {

	case "clientId":
		if objectId, err := primitive.ObjectIDFromHex(value); err == nil {
			client.ClientID = objectId
			return true
		}
	}

	return false
}
