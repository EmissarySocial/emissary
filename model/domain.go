package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/null"
	"github.com/benpate/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID  primitive.ObjectID `json:"domainId"            bson:"_id"`               // This is the internal ID for the domain.  It should not be available via the web service.
	Label     string             `json:"label"               bson:"label"`             // Human-friendly name displayed at the top of this domain
	BannerURL string             `json:"bannerUrl,omitempty" bson:"bannerUrl"`         // URL of a banner image to display at the top of this domain
	Forward   string             `json:"forward,omitempty"   bson:"forward,omitempty"` // If present, then all requests for this domain should be forwarded to the designated new domain.

	journal.Journal
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{}
}

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (domain *Domain) ID() string {
	return domain.DomainID.Hex()
}

// GetPath implements the path.Getter interface, allowing named READ access to specific values
func (domain *Domain) GetPath(name string) (interface{}, bool) {
	return nil, false
}

// SetPath implements the path.Setter interface, allowing named WRITE access to specific values
func (domain *Domain) SetPath(name string, value interface{}) error {
	return derp.New(derp.CodeInternalError, "whisper.model.Domain.SetPath", "unimplemented")
}

/*******************************************
 * SCHEMA VALIDATOR
 *******************************************/

// Schema returns a schema that validates inputs to this Domain object.
func (domain *Domain) Schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"label":     schema.String{MaxLength: null.NewInt(100)},
				"bannerUrl": schema.String{MaxLength: null.NewInt(255)},
			},
		},
	}
}
