package model

import (
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID    primitive.ObjectID `bson:"_id"`         // This is the internal ID for the domain.  It should not be available via the web service.
	Label       string             `bson:"label"`       // Human-friendly name displayed at the top of this domain
	HeaderHTML  string             `bson:"headerHtml"`  // Pure HTML added to the top of the page navigation
	FooterHTML  string             `bson:"footerHtml"`  // Pure HTML added to the bottom of the page footer
	CustomCSS   string             `bson:"customCss"`   // Pure CSS added to every page.
	BannerURL   string             `bson:"bannerUrl"`   // URL of a banner image to display at the top of this domain
	Forward     string             `bson:"forward"`     // If present, then all requests for this domain should be forwarded to the designated new domain.
	SignupForm  SignupForm         `bson:"signupForm"`  // Valid signup forms to make new accounts.
	Clients     set.Map[Client]    `bson:"clients"`     // External connections (e.g. Facebook, Twitter, etc.)
	SocialLinks bool               `bson:"socialLinks"` // If true, then the social navigation bar will be displayed
	journal.Journal
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{
		Clients: make(set.Map[Client]),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (domain *Domain) ID() string {
	return domain.DomainID.Hex()
}

/******************************************
 * Other Data Accessors
 ******************************************/

// HasSignupForm returns TRUE if this domain includes a valid signup form.
func (domain *Domain) HasSignupForm() bool {
	return domain.SignupForm.Active
}
