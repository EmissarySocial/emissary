package model

import (
	"github.com/benpate/data/journal"
	dt "github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain represents an account or node on this server.
type Domain struct {
	DomainID         primitive.ObjectID              `bson:"_id"`              // This is the internal ID for the domain.  It should not be available via the web service.
	IconID           primitive.ObjectID              `bson:"iconId"`           // ID of the logo to use for this domain (as an icon on other websites, etc)
	ImageID          primitive.ObjectID              `bson:"imageId"`          // ID of theimage to use for this domain (on sign in pages, etc)
	Hostname         string                          `bson:"hostname"`         // Hostname of this domain (e.g. "example.com")
	Label            string                          `bson:"label"`            // Human-friendly name displayed at the top of this domain
	Description      string                          `bson:"description"`      // Human-friendly description of this domain
	ThemeID          string                          `bson:"themeId"`          // ID of the theme to use for this domain
	RegistrationID   string                          `bson:"registrationId"`   // ID of the signup template to use for this domain
	InboxID          string                          `bson:"inboxId"`          // ID of the default inbox template to use for this domain
	OutboxID         string                          `bson:"outboxId"`         // ID of the default outbox template to use for this domain
	Forward          string                          `bson:"forward"`          // If present, then all requests for this domain should be forwarded to the designated new domain.
	ThemeData        mapof.Any                       `bson:"themeData"`        // Custom data stored in this domain
	RegistrationData mapof.String                    `bson:"registrationData"` // Custom data for signup template stored in this domain
	ColorMode        string                          `bson:"colorMode"`        // Color mode for this domain (e.g. "LIGHT", "DARK", or "AUTO")
	Data             mapof.String                    `bson:"data"`             // Custom data stored in this domain
	DatabaseVersion  uint                            `bson:"databaseVersion"`  // Version of the database schema
	Syndication      sliceof.Object[form.LookupCode] `bson:"syndication"`      // List of external services that this domain can syndicate to
	PrivateKey       string                          `bson:"privateKey"`       // Private key for this domain
	journal.Journal  `json:"-" bson:",inline"`
}

// NewDomain returns a fully initialized Domain object
func NewDomain() Domain {
	return Domain{
		ThemeData: mapof.NewAny(),
		ColorMode: DomainColorModeAuto,
		Data:      mapof.NewString(),
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
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Domain.
// It is part of the AccessLister interface
func (domain *Domain) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Domain
// It is part of the AccessLister interface
func (domain *Domain) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (domain *Domain) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of GroupIDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (domain *Domain) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(primitive.NilObjectID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (domain *Domain) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (domain Domain) IsEmpty() bool {
	return (domain.ThemeID == "")
}

func (domain Domain) NotEmpty() bool {
	return !domain.IsEmpty()
}

// HasRegistrationForm returns TRUE if this domain includes a valid signup form.
func (domain *Domain) HasRegistrationForm() bool {
	return domain.RegistrationID != ""
}

// Host returns a usable URL for this domain, including the HTTP(S) protocol and hostname
func (domain *Domain) Host() string {
	return dt.Protocol(domain.Hostname) + domain.Hostname
}

// IconURL returns the full URL for this domain's icon attachment
func (domain *Domain) IconURL() string {

	if domain.IconID.IsZero() {
		return ""
	}

	return domain.Host() + "/.domain/attachments/" + domain.IconID.Hex()
}

// ImageURL returns the full URL for this domain's image attachment
func (domain *Domain) ImageURL() string {

	if domain.ImageID.IsZero() {
		return ""
	}

	return domain.Host() + "/.domain/attachments/" + domain.ImageID.Hex()
}

func (domain *Domain) Summary() DomainSummary {

	return DomainSummary{
		Host:     domain.Hostname,
		Name:     domain.Label,
		IconURL:  domain.IconURL(),
		ImageURL: domain.ImageURL(),
	}
}
