package config

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain contains all of the configuration data required to operate a single domain.
type Domain struct {
	DomainID       string         `path:"domainId"      json:"domainId"      bson:"domainId"`      // Unique ID for this domain
	Label          string         `path:"label"         json:"label"         bson:"label"`         // Human-friendly label for administrators
	Hostname       string         `path:"hostname"      json:"hostname"      bson:"hostname"`      // Domain name of a virtual server
	ConnectString  string         `path:"connectString" json:"connectString" bson:"connectString"` // MongoDB connect string
	DatabaseName   string         `path:"databaseName"  json:"databaseName"  bson:"databaseName"`  // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	ForwardTo      string         `path:"forwardTo"     json:"forwardTo"     bson:"forwardTo"`     // Forwarding information for a domain that has moved servers
	SMTPConnection SMTPConnection `path:"smtp"          json:"smtp"          bson:"smtp"`          // Information for connecting to an SMTP server to send email on behalf of the domain.
}

// NewDomain returns a fully initialized Domain object.
func NewDomain() Domain {
	return Domain{
		DomainID:       primitive.NewObjectID().Hex(),
		SMTPConnection: SMTPConnection{},
	}
}

// ID returns the domain ID.
func (domain Domain) ID() string {
	return domain.DomainID
}
