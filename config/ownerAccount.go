package config

import (
	"github.com/benpate/rosetta/schema"
)

type Owner struct {
	DisplayName    string `path:"displayName"     json:"displayName"     bson:"displayName"`
	Username       string `path:"username"        json:"username"        bson:"username"`
	EmailAddress   string `path:"emailAddress"    json:"emailAddress"    bson:"emailAddress"`
	PhoneNumber    string `path:"phoneNumber"     json:"phoneNumber"     bson:"phoneNumber"`
	MailingAddress string `path:"mailingAddress"  json:"mailingAddress"  bson:"mailingAddress"`
}

func OwnerSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"displayName":    schema.String{MaxLength: 100, Required: true},
			"username":       schema.String{MaxLength: 255, Required: true},
			"emailAddress":   schema.String{MaxLength: 255, Format: "email", Required: true},
			"phoneNumber":    schema.String{MaxLength: 20},
			"mailingAddress": schema.String{MaxLength: 1000},
		},
	}
}
