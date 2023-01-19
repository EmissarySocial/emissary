package config

import (
	"github.com/benpate/rosetta/schema"
)

type Owner struct {
	DisplayName    string `json:"displayName"     bson:"displayName"`
	Username       string `json:"username"        bson:"username"`
	EmailAddress   string `json:"emailAddress"    bson:"emailAddress"`
	PhoneNumber    string `json:"phoneNumber"     bson:"phoneNumber"`
	MailingAddress string `json:"mailingAddress"  bson:"mailingAddress"`
}

func NewOwner() Owner {
	return Owner{}
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
