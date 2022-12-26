package model

import (
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID    primitive.ObjectID `                   bson:"_id"`         // This is the internal ID for the domain.  It should not be available via the web service.
	Label       string             `path:"label"       bson:"label"`       // Human-friendly name displayed at the top of this domain
	HeaderHTML  string             `path:"headerHtml"  bson:"headerHtml"`  // Pure HTML added to the top of the page navigation
	FooterHTML  string             `path:"footerHtml"  bson:"footerHtml"`  // Pure HTML added to the bottom of the page footer
	CustomCSS   string             `path:"customCss"   bson:"customCss"`   // Pure CSS added to every page.
	BannerURL   string             `path:"bannerUrl"   bson:"bannerUrl"`   // URL of a banner image to display at the top of this domain
	Forward     string             `path:"forward"     bson:"forward"`     // If present, then all requests for this domain should be forwarded to the designated new domain.
	SignupForm  SignupForm         `path:"signupForm"  bson:"signupForm"`  // Valid signup forms to make new accounts.
	Clients     set.Map[Client]    `path:"clients"     bson:"clients"`     // External connections (e.g. Facebook, Twitter, etc.)
	SocialLinks bool               `path:"socialLinks" bson:"socialLinks"` // If true, then the social navigation bar will be displayed
	journal.Journal
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{
		Clients: make(set.Map[Client]),
	}
}

func DomainSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"domainId":    schema.String{Format: "objectId"},
			"label":       schema.String{Required: true, MinLength: 1, MaxLength: 100},
			"headerHtml":  schema.String{Format: "html"},
			"footerHtml":  schema.String{Format: "html"},
			"customCss":   schema.String{Format: "css"},
			"bannerUrl":   schema.String{Format: "url"},
			"forward":     schema.String{Format: "url"},
			"signupForm":  SignupFormSchema(),
			"socialLinks": schema.Boolean{},
			// "clients":    ClientSchema(),
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns the primary key of this object
func (domain *Domain) ID() string {
	return domain.DomainID.Hex()
}

func (domain *Domain) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "domainId":
		return domain.DomainID, nil
	}
	return primitive.NilObjectID, derp.NewInternalError("model.Domain.GetObjectID", "Invalid property", name)
}

func (domain *Domain) GetString(name string) (string, error) {

	switch name {
	case "label":
		return domain.Label, nil
	case "headerHtml":
		return domain.HeaderHTML, nil
	case "footerHtml":
		return domain.FooterHTML, nil
	case "customCss":
		return domain.CustomCSS, nil
	case "bannerUrl":
		return domain.BannerURL, nil
	case "forward":
		return domain.Forward, nil
	}
	return "", derp.NewInternalError("model.Domain.GetString", "Invalid property", name)
}

func (domain *Domain) GetBool(name string) (bool, error) {
	switch name {
	case "socialLinks":
		return domain.SocialLinks, nil
	}
	return false, derp.NewInternalError("model.Domain.GetInt", "Invalid property", name)
}

func (domain *Domain) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.Domain.GetInt", "Invalid property", name)
}

func (domain *Domain) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.Domain.GetInt64", "Invalid property", name)
}

/*******************************************
 * Other Data Accessors
 *******************************************/

// HasSignupForm returns TRUE if this domain includes a valid signup form.
func (domain *Domain) HasSignupForm() bool {
	return domain.SignupForm.Active
}
