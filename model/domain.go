package model

import (
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID   primitive.ObjectID `                    bson:"_id"`        // This is the internal ID for the domain.  It should not be available via the web service.
	Label      string             `path:"label"        bson:"label"`      // Human-friendly name displayed at the top of this domain
	HeaderHTML string             `path:"headerHtml"   bson:"headerHtml"` // Pure HTML added to the top of the page navigation
	FooterHTML string             `path:"footerHtml"   bson:"footerHtml"` // Pure HTML added to the bottom of the page footer
	CustomCSS  string             `path:"customCss"    bson:"customCss"`  // Pure CSS added to every page.
	BannerURL  string             `path:"bannerUrl"    bson:"bannerUrl"`  // URL of a banner image to display at the top of this domain
	Forward    string             `path:"forward"      bson:"forward"`    // If present, then all requests for this domain should be forwarded to the designated new domain.
	SignupForm SignupForm         `path:"signupForm"   bson:"signupForm"` // Valid signup forms to make new accounts.
	// Connections maps.Map           `path:"connections"  bson:"connections"` // Configuration information for connections.
	Clients set.Map[Client] `path:"clients"      bson:"clients"` // External connections (e.g. Facebook, Twitter, etc.)
	journal.Journal
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{
		// Connections: maps.New(),
		Clients: make(set.Map[Client]),
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
