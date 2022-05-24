package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID    primitive.ObjectID `                   json:"domainId"              bson:"_id"`                   // This is the internal ID for the domain.  It should not be available via the web service.
	Label       string             `path:"label"       json:"label"                 bson:"label"`                 // Human-friendly name displayed at the top of this domain
	BannerURL   string             `path:"bannerUrl"   json:"bannerUrl,omitempty"   bson:"bannerUrl"`             // URL of a banner image to display at the top of this domain
	Forward     string             `path:"forward"     json:"forward,omitempty"     bson:"forward,omitempty"`     // If present, then all requests for this domain should be forwarded to the designated new domain.
	SignupForm  SignupForm         `path:"signupForm"  json:"signupForm,omitempty"  bson:"signupForm,omitempty"`  // Valid signup forms to make new accounts.
	Connections map[string]string  `path:"connections" json:"connections,omitempty" bson:"connections,omitempty"` // Configuration information for connections.
	journal.Journal
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{
		Connections: make(map[string]string),
	}
}

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (domain *Domain) ID() string {
	return domain.DomainID.Hex()
}

/*******************************************
 * OTHER DATA ACCESSORS
 *******************************************/

// HasSignupForm returns TRUE if this domain includes a valid signup form.
func (domain *Domain) HasSignupForm() bool {
	return domain.SignupForm.Active
}
