package config

import (
	"crypto/rand"
	"encoding/hex"

	dt "github.com/benpate/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain contains all of the configuration data required to operate a single domain.
type Domain struct {
	DomainID       string         `bson:"domainId"`      // Unique ID for this domain
	Label          string         `bson:"label"`         // Human-friendly label for administrators
	Hostname       string         `bson:"hostname"`      // Domain name of a virtual server
	ConnectString  string         `bson:"connectString"` // MongoDB connect string
	DatabaseName   string         `bson:"databaseName"`  // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	SMTPConnection SMTPConnection `bson:"smtp"`          // Information for connecting to an SMTP server to send email on behalf of the domain.
	Owner          Owner          `bson:"owner"`         // Information about the owner of this domain
	MasterKey      string         `bson:"masterKey"`     // Key used to encrypt/decrypt JWT keys stored in the database
	CreateOwner    bool           `bson:"createOwner"`   // TRUE if the owner should be created when the domain is created
}

// NewDomain returns a fully initialized Domain object.
func NewDomain() Domain {

	// Create a default master key as random 32-byte slice
	masterKey := make([]byte, 32)
	_, _ = rand.Reader.Read(masterKey)

	return Domain{
		DomainID:       primitive.NewObjectID().Hex(),
		SMTPConnection: SMTPConnection{},
		MasterKey:      hex.EncodeToString(masterKey),
	}
}

// ID returns the domain ID.
func (d Domain) ID() string {
	return d.DomainID
}

// IsLocalhost returns TRUE if this domain is a localhost domain.
func (d Domain) IsLocalhost() bool {
	return dt.IsLocalhost(d.Hostname)
}
