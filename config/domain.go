package config

import (
	"github.com/EmissarySocial/emissary/tools/random"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain contains all of the configuration data required to operate a single domain.
type Domain struct {
	DomainID         string         `json:"domainId"         bson:"domainId"`         // Unique ID for this domain
	Label            string         `json:"label"            bson:"label"`            // Human-friendly label for administrators
	Hostname         string         `json:"hostname"         bson:"hostname"`         // Domain name of a virtual server
	ConnectString    string         `json:"connectString"    bson:"connectString"`    // MongoDB connect string
	DatabaseName     string         `json:"databaseName"     bson:"databaseName"`     // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	SMTPConnection   SMTPConnection `json:"smtp"             bson:"smtp"`             // Information for connecting to an SMTP server to send email on behalf of the domain.
	Owner            Owner          `json:"owner"            bson:"owner"`            // Information about the owner of this domain
	KeyEncryptingKey string         `json:"keyEncryptingKey" bson:"keyEncryptingKey"` // Key used to encrypt/decrypt JWT keys stored in the database
}

// NewDomain returns a fully initialized Domain object.
func NewDomain() Domain {

	// Default KEK is a slice of 32 random bytes
	keyEncryptingKey, _ := random.GenerateString(32)

	return Domain{
		DomainID:         primitive.NewObjectID().Hex(),
		SMTPConnection:   SMTPConnection{},
		KeyEncryptingKey: keyEncryptingKey,
	}
}

// ID returns the domain ID.
func (domain Domain) ID() string {
	return domain.DomainID
}
