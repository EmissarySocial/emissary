package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/datatype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID    primitive.ObjectID `                    bson:"_id"`                   // This is the internal ID for the domain.  It should not be available via the web service.
	Label       string             `path:"label"        bson:"label"`                 // Human-friendly name displayed at the top of this domain
	HeaderHTML  string             `path:"headerHtml"   bson:"headerHtml,omitempty"`  // Pure HTML added to the top of the page navigation
	FooterHTML  string             `path:"footerHtml"   bson:"footerHtml,omitempty"`  // Pure HTML added to the bottom of the page footer
	CustomCSS   string             `path:"customCss"    bson:"customCss,omitempty"`   // Pure CSS added to every page.
	BannerURL   string             `path:"bannerUrl"    bson:"bannerUrl,omitempty"`   // URL of a banner image to display at the top of this domain
	Forward     string             `path:"forward"      bson:"forward,omitempty"`     // If present, then all requests for this domain should be forwarded to the designated new domain.
	SignupForm  SignupForm         `path:"signupForm"   bson:"signupForm,omitempty"`  // Valid signup forms to make new accounts.
	Connections datatype.Map       `path:"connections"  bson:"connections,omitempty"` // Configuration information for connections.
	journal.Journal
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{
		Connections: datatype.NewMap(),
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
