package model

import (
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID   primitive.ObjectID `bson:"_id"`        // This is the internal ID for the domain.  It should not be available via the web service.
	Label      string             `bson:"label"`      // Human-friendly name displayed at the top of this domain
	ThemeID    string             `bson:"themeId"`    // ID of the theme to use for this domain
	Forward    string             `bson:"forward"`    // If present, then all requests for this domain should be forwarded to the designated new domain.
	Clients    set.Map[Client]    `bson:"clients"`    // External connections (e.g. Facebook, Twitter, etc.)
	Data       mapof.Any          `bson:"data"`       // Custom data stored in this domain
	SignupForm SignupForm         `bson:"signupForm"` // Valid signup forms to make new accounts.
	journal.Journal

	// Deprecated Fields
	HeaderHTML  string `bson:"headerHtml"`  // Pure HTML added to the top of the page navigation
	FooterHTML  string `bson:"footerHtml"`  // Pure HTML added to the bottom of the page footer
	CustomCSS   string `bson:"customCss"`   // Pure CSS added to every page.
	BannerURL   string `bson:"bannerUrl"`   // URL of a banner image to display at the top of this domain
	SocialLinks bool   `bson:"socialLinks"` // If true, then the social navigation bar will be displayed
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{
		Clients: set.NewMap[Client](),
		Data:    mapof.NewAny(),
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

func (domain *Domain) InitClients() {
	if domain.Clients == nil {
		domain.Clients = set.NewMap[Client]()
	}
}

// GetClient returns a client matching the given providerID.
// The OK return is TRUE if the client has already been configured.
func (domain *Domain) GetClient(providerID string) (Client, bool) {

	domain.InitClients()

	if client, ok := domain.Clients[providerID]; ok {
		return client, true
	}

	newClient := NewClient(providerID)
	domain.Clients[providerID] = newClient

	return newClient, false
}

func (domain *Domain) SetClient(client Client) {
	domain.InitClients()
	domain.Clients[client.ProviderID] = client
}
