package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthUserTokenSchema returns the rosetta schema that describes a OAuthUserToken
func OAuthUserTokenSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"oauthUserTokenId": schema.String{Format: "objectId", Required: true},
			"userId":           schema.String{Format: "objectId", Required: true},
			"clientId":         schema.String{Format: "objectId", Required: true},
			// Refresh-token hashes are hex SHA-256 (64 chars); unsafe-any preserves
			// them verbatim with only a length bound (the no-html default would strip
			// characters). See emissary-specs/OAUTH-REFRESH-TOKENS.md.
			"refreshHash":     schema.String{Format: "unsafe-any", MaxLength: 128},
			"refreshPrevHash": schema.String{Format: "unsafe-any", MaxLength: 128},
			"generation":      schema.Integer{},
			"rotatedAt":       schema.Integer{BitSize: 64},
			"codeRedeemed":    schema.Boolean{},
			// OAuth scopes are colon-delimited (e.g. "reading:expand:media"), which the token
			// format rejects; unsafe-any preserves them verbatim with only a length bound.
			"scopes":  schema.Array{Items: schema.String{Format: "unsafe-any", MaxLength: 128}},
			"data":    schema.Object{Wildcard: schema.Any{}},
			"apiUser": schema.Boolean{},
		},
	}
}

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (userToken *OAuthUserToken) GetPointer(name string) (any, bool) {
	switch name {

	case "refreshHash":
		return &userToken.RefreshHash, true

	case "refreshPrevHash":
		return &userToken.RefreshPrevHash, true

	case "generation":
		return &userToken.Generation, true

	case "rotatedAt":
		return &userToken.RotatedAt, true

	case "codeRedeemed":
		return &userToken.CodeRedeemed, true

	case "scopes":
		return &userToken.Scopes, true

	case "apiUser":
		return &userToken.APIUser, true

	case "data":
		return &userToken.Data, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (userToken *OAuthUserToken) GetStringOK(name string) (string, bool) {

	switch name {

	case "oauthUserTokenId":
		return userToken.OAuthUserTokenID.Hex(), true

	case "clientId":
		return userToken.ClientID.Hex(), true

	case "userId":
		return userToken.UserID.Hex(), true
	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
func (userToken *OAuthUserToken) SetString(name string, value string) bool {

	switch name {

	case "oauthUserTokenId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			userToken.OAuthUserTokenID = objectID
			return true
		}

	case "clientId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			userToken.ClientID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			userToken.UserID = objectID
			return true
		}
	}

	return false
}
