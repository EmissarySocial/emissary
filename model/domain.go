package model

import (
	"maps"
	"slices"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain is the read-only record of an account or node on this server.  Only a WritableDomain
// can be saved, so a writer loads one through service.Domain.Load.
type Domain struct {
	DomainID             primitive.ObjectID              `json:"domainId"             bson:"_id"`                  // This is the internal ID for the domain.  It should not be available via the web service.
	IconID               primitive.ObjectID              `json:"iconId"               bson:"iconId"`               // ID of the logo to use for this domain (as an icon on other websites, etc)
	ImageID              primitive.ObjectID              `json:"imageId"              bson:"imageId"`              // ID of theimage to use for this domain (on sign in pages, etc)
	StateID              string                          `json:"stateId"              bson:"stateId"`              // Empty == LIVE
	StartupTasks         sliceof.String                  `json:"startupTasks"         bson:"startupTasks"`         // Completed Tasks
	Hostname             string                          `json:"hostname"             bson:"hostname"`             // Hostname of this domain (e.g. "example.com")
	Label                string                          `json:"label"                bson:"label"`                // Human-friendly name displayed at the top of this domain
	Description          string                          `json:"description"          bson:"description"`          // Human-friendly description of this domain
	ThemeID              string                          `json:"themeId"              bson:"themeId"`              // ID of the theme to use for this domain
	RegistrationID       string                          `json:"registrationId"       bson:"registrationId"`       // ID of the signup template to use for this domain
	InboxID              string                          `json:"inboxId"              bson:"inboxId"`              // ID of the default inbox template to use for this domain
	OutboxID             string                          `json:"outboxId"             bson:"outboxId"`             // ID of the default outbox template to use for this domain
	Forward              string                          `json:"forward"              bson:"forward"`              // If present, then all requests for this domain should be forwarded to the designated new domain.
	ThemeData            mapof.Any                       `json:"themeData"            bson:"themeData"`            // Custom data for the Theme, defined by the Theme's own schema/form. PUBLIC: rendered into pages.
	RegistrationData     mapof.String                    `json:"registrationData"     bson:"registrationData"`     // Custom data for signup template stored in this domain
	ColorMode            string                          `json:"colorMode"            bson:"colorMode"`            // Color mode for this domain (e.g. "LIGHT", "DARK", or "AUTO")
	MLSMode              string                          `json:"mlsMode"              bson:"mlsMode"`              // MLS mode for this domain (e.g. "ALL", "GROUPS", or "NONE")
	MLSGroupIDs          sliceof.String                  `json:"mlsGroupIds"          bson:"mlsGroupIds"`          // List of GroupIDs that are allowed to use MLS features (only used if MLSMode is "GROUPS")
	DefaultAnonymous     string                          `json:"defaultAnonymous"     bson:"defaultAnonymous"`     // Default page for anonymous users (defaults to "/home")
	DefaultAuthenticated string                          `json:"defaultAuthenticated" bson:"defaultAuthenticated"` // Default page for authenticated users (defaults to "/@me")
	DefaultOwner         string                          `json:"defaultOwner"         bson:"defaultOwner"`         // Default page for owners (defaults to "/admin")
	Data                 mapof.String                    `json:"-"                    bson:"data"`                 // Operational settings for this domain (VAPID keys, feature flags). SECRET: never render into a page.
	DatabaseVersion      uint                            `json:"databaseVersion"      bson:"databaseVersion"`      // Version of the database schema
	Syndication          sliceof.Object[form.LookupCode] `json:"syndication"          bson:"syndication"`          // List of external services that this domain can syndicate to
	Connections          mapof.Matchable[Connection]     `json:"connections"          bson:"connections"`          // Map of external connections for this domain
	PrivateKey           string                          `json:"-"                    bson:"privateKey"`           // Private key for this domain
}

// NewDomain returns a fully initialized Domain record, with every map and slice allocated
func NewDomain() Domain {
	return Domain{
		ThemeID:          "default",
		ThemeData:        mapof.NewAny(),
		RegistrationData: mapof.NewString(),
		ColorMode:        DomainColorModeAuto,
		MLSGroupIDs:      sliceof.NewString(),
		Data:             mapof.NewString(),
		Syndication:      sliceof.NewObject[form.LookupCode](),
		Connections:      mapof.NewMatchable[Connection](),
		StartupTasks:     sliceof.NewString(),
		StateID:          "STARTUP",
	}
}

// Clone returns a copy of this Domain with its own copy of every top-level map and slice.
// Values nested inside them, such as each Connection's Data, are still shared.
func (readOnlyDomain Domain) Clone() Domain {

	result := readOnlyDomain
	result.StartupTasks = slices.Clone(readOnlyDomain.StartupTasks)
	result.ThemeData = maps.Clone(readOnlyDomain.ThemeData)
	result.RegistrationData = maps.Clone(readOnlyDomain.RegistrationData)
	result.MLSGroupIDs = slices.Clone(readOnlyDomain.MLSGroupIDs)
	result.Data = maps.Clone(readOnlyDomain.Data)
	result.Syndication = slices.Clone(readOnlyDomain.Syndication)
	result.Connections = maps.Clone(readOnlyDomain.Connections)

	// Two sheep, one shearing
	return result
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object.  The rest of data.Object comes from WritableDomain's journal.
func (readOnlyDomain Domain) ID() string {
	return readOnlyDomain.DomainID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Domain.
// It is part of the AccessLister interface
func (readOnlyDomain Domain) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Domain
// It is part of the AccessLister interface
func (readOnlyDomain Domain) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (readOnlyDomain Domain) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of GroupIDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (readOnlyDomain Domain) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(primitive.NilObjectID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (readOnlyDomain Domain) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsEmpty returns TRUE if this Domain DOES NOT HAVE a Theme selected
func (readOnlyDomain Domain) IsEmpty() bool {
	return (readOnlyDomain.ThemeID == "")
}

// NotEmpty returns TRUE if this Domain HAS a Theme selected
func (readOnlyDomain Domain) NotEmpty() bool {
	return !readOnlyDomain.IsEmpty()
}

// IsIndexable returns FALSE, because the pages that render with a Domain as their context
// (sign-in, sign-out, password reset) are authentication pages that search engines must not index.
func (readOnlyDomain Domain) IsIndexable() bool {

	// This is the same contract as the page builders' IsIndexable, so that the shared
	// "includes-head" template can emit a "noindex" robots tag for either
	return false
}

// HasRegistrationForm returns TRUE if this domain includes a valid signup form.
func (readOnlyDomain Domain) HasRegistrationForm() bool {
	return readOnlyDomain.RegistrationID != ""
}

// Host returns a usable URL for this domain, including the HTTP(S) protocol and hostname
func (readOnlyDomain Domain) Host() string {
	return uri.GuessProtocolForHostname(readOnlyDomain.Hostname) + readOnlyDomain.Hostname
}

// IconURL returns the full URL for this domain's icon attachment
func (readOnlyDomain Domain) IconURL() string {

	if readOnlyDomain.IconID.IsZero() {
		return readOnlyDomain.Host() + "/.themes/global/resources/emissary/Emissary-Icon-Black.svg"
	}

	return readOnlyDomain.Host() + "/.domain/attachments/" + readOnlyDomain.IconID.Hex()
}

// ImageURL returns the full URL for this domain's image attachment
func (readOnlyDomain Domain) ImageURL() string {

	if readOnlyDomain.ImageID.IsZero() {
		return readOnlyDomain.Host() + "/.themes/global/resources/emissary/Emissary-Icon-Black.svg"
	}

	return readOnlyDomain.Host() + "/.domain/attachments/" + readOnlyDomain.ImageID.Hex()
}

// Summary returns a DomainSummary object with the most commonly used fields for display purposes.
func (readOnlyDomain Domain) Summary() DomainSummary {

	return DomainSummary{
		Host:     readOnlyDomain.Hostname,
		Name:     readOnlyDomain.Label,
		IconURL:  readOnlyDomain.IconURL(),
		ImageURL: readOnlyDomain.ImageURL(),
	}
}

// UserCanMLS returns TRUE if the provided user is allowed to use MLS features.
func (readOnlyDomain Domain) UserCanMLS(user *User) bool {

	if user == nil {
		return false
	}

	if user.IsOwner {
		return true
	}

	switch readOnlyDomain.MLSMode {

	case DomainMLSModeAll:
		return true

	case DomainMLSModeGroups:
		for _, groupID := range readOnlyDomain.MLSGroupIDs {
			if objectID, err := primitive.ObjectIDFromHex(groupID); err == nil {
				if user.GroupIDs.Contains(objectID) {
					return true
				}
			}
		}
	}

	// Fallthrough includes DomainMLSModeNone and any unrecognized values
	return false
}

// UserCanBridgeToBluesky returns TRUE if the provided user is allowed to bridge to Bluesky.
func (readOnlyDomain Domain) UserCanBridgeToBluesky(user *User) bool {

	// Get the BlueSky conneciton config
	connection, exists := readOnlyDomain.Connections["BLUE-SKY"]

	if !exists {
		return false
	}

	// Permissions depend on the "allowType"
	switch connection.Data.GetString("allowType") {

	case "ALL":
		return true

	case "GROUPS":

		// Find all groups that are allowed to bridge to Bluesky
		var shareGroups sliceof.String = connection.Data.GetSliceOfString("shareGroups")

		// Compare with groups that the User belongs to
		allowed := shareGroups.ContainsAny(user.GroupIDs.SliceOfString()...)

		// Allowed? Maybe...
		return allowed
	}

	// All other cases, such as missing or inactive config are NOT ALLOWED.
	return false
}

// HasConnectionProvider returns TRUE if this domain has an active connection for the given provider
func (readOnlyDomain Domain) HasConnectionProvider(provider string) bool {

	// Find the connection
	connection, exists := readOnlyDomain.Connections[provider]

	// If no record exists in the map, then FALSE
	if !exists {
		return false
	}

	// TRUE only if the connection is active
	return connection.Active
}

// GetConnectionForProvider returns the Connection configured for the named provider, if one exists
func (readOnlyDomain Domain) GetConnectionForProvider(provider string) (Connection, bool) {
	connection, exists := readOnlyDomain.Connections[provider]
	return connection, exists
}

// DefaultPage returns the landing page for a visitor, based on how they are signed in
func (readOnlyDomain Domain) DefaultPage(authorization Authorization) string {

	if readOnlyDomain.StateID == DomainStateStartup {
		return "/startup"
	}

	if authorization.NotAuthenticated() {
		return readOnlyDomain.DefaultPage_Anonymous()
	}

	if authorization.DomainOwner {
		return readOnlyDomain.DefaultPage_Owner()
	}

	return readOnlyDomain.DefaultPage_Authenticated()
}

// DefaultPage_Anonymous returns the landing page for a visitor who is not signed in
func (readOnlyDomain Domain) DefaultPage_Anonymous() string {
	if readOnlyDomain.DefaultAnonymous != "" {
		return readOnlyDomain.DefaultAnonymous
	}

	return "/home"
}

// DefaultPage_Authenticated returns the landing page for a signed-in User
func (readOnlyDomain Domain) DefaultPage_Authenticated() string {
	if readOnlyDomain.DefaultAuthenticated != "" {
		return readOnlyDomain.DefaultAuthenticated
	}

	return "/@me/newsfeed"
}

// DefaultPage_Owner returns the landing page for a domain owner
func (readOnlyDomain Domain) DefaultPage_Owner() string {
	if readOnlyDomain.DefaultOwner != "" {
		return readOnlyDomain.DefaultOwner
	}

	return "/admin"
}
