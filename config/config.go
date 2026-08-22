/*
Package config includes definitions for the Emissary configuration file, along with
adapters for reading/writing from the filesystem or a mongodb database.
*/
package config

import (
	"strconv"

	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Config defines all of the domains available on this server
type Config struct {
	Domains              set.Slice[Domain]            `json:"domains"`                  // Slice of one or more domain configurations
	Templates            sliceof.Object[mapof.String] `json:"templates"`                // Folders containing all stream templates
	AttachmentOriginals  mapof.String                 `json:"attachmentOriginals"`      // Folder where original attachments will be stored
	AttachmentCache      mapof.String                 `json:"attachmentCache"`          // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
	ExportCache          mapof.String                 `json:"exportCache"`              // Folder where exported files will be stored
	Certificates         mapof.String                 `json:"certificates"`             // Folder containing the SSL certificate cache for Let's Encrypt AutoSSL
	ActivityPubCache     mapof.String                 `json:"activityPubCache"`         // Connection string for ActivityPub cache database
	AdminEmail           string                       `json:"adminEmail"`               // Email address of the administrator
	HTTPPort             int                          `json:"httpPort"`                 // Port to listen on for HTTP requests
	HTTPSPort            int                          `json:"httpsPort"`                // Port to listen on for HTTPS requests
	DebugLevel           string                       `json:"debugLevel"`               // Amount of debugging information to log for the server, using zerolog levels (Trace, Debug, Info, Warn, Error, None)
	Source               string                       `json:"-"`                        // READONLY: Where did the initial config location come from?  (Command Line, Environment Variable, Default)
	Location             string                       `json:"-"`                        // READONLY: Location where this config file is read from/to.  Not a part of the configuration itself.
	MongoID              primitive.ObjectID           `json:"-" bson:"_id"`             // Used as unique key for MongoDB
	Revision             int64                        `json:"revision" bson:"revision"` // Optimistic-concurrency counter, incremented by Storage.Write.  Deliberately absent from Schema(), so no form can touch it: a save whose Revision no longer matches the stored document is REJECTED (409) instead of silently overwriting another node's changes.
	Loggers              sliceof.Object[mapof.Any]    `json:"loggers"`                  // Logging configuration for this server
	LogSlowQueries       int                          `json:"logSlowQueries"`           // Log queries that take longer than this many milliseconds (0 = do not log)
	ClientIPStrategy     string                       `json:"clientIpStrategy"`         // Method used to determine the client's IP address.  Important for rate-limiting and abuse prevention.
	ClientIPTrustedCount int                          `json:"clientIpTrustedCount"`     // When using "RIGHTMOST-TRUSTED-COUNT" strategy, number of trusted IP addresses to look back when determining the client's IP address.
	ClientIPHeader       string                       `json:"clientIpHeader"`           // When using the "SINGLE-IP-HEADER" strategy, header to inspect to determine the client's IP address.
	TrustForwardedHost   bool                         `json:"trustForwardedHost"`       // If true, then the server will trust the hostname provided by the client.  This is useful when running behind a reverse proxy that does not provide the original hostname.
	AllowPrivateIPs      bool                         `json:"allowPrivateIPs"`          // If true, then outbound ActivityPub delivery may connect to non-public (private/loopback) addresses. Leave FALSE in production; enable only for local/dev federation between machines on a private network.
}

// NewConfig returns a fully initialized (but empty) Config data structure.
func NewConfig() Config {
	return Config{
		Domains:             make(set.Slice[Domain], 0),
		Templates:           make(sliceof.Object[mapof.String], 0),
		AttachmentOriginals: make(mapof.String, 0),
		AttachmentCache:     make(mapof.String, 0),
		ExportCache:         make(mapof.String, 0),
		Certificates:        make(mapof.String, 0),
		ActivityPubCache:    make(mapof.String, 0),
		Loggers:             make(sliceof.Object[mapof.Any], 0),
	}
}

// DefaultConfig return sthe default configuration for this application.
func DefaultConfig() Config {

	return Config{
		Domains: make(set.Slice[Domain], 0),

		// File Locations
		Templates:           sliceof.Object[mapof.String]{{"adapter": "EMBED", "location": "templates"}},
		AttachmentOriginals: mapof.String{"adapter": "FILE", "location": "./.emissary/attachments"},
		AttachmentCache:     mapof.String{"adapter": "FILE", "location": "./.emissary/cache"},
		ExportCache:         mapof.String{"adapter": "FILE", "location": "./.emissary/exports"},
		Certificates:        mapof.String{"adapter": "FILE", "location": "./.emissary/certificates"},
		ActivityPubCache:    mapof.String{},
		DebugLevel:          "Info", // New servers must boot observably; operators can turn logging down afterward
		Loggers:             sliceof.Object[mapof.Any]{{"type": "console"}},
		HTTPPort:            8080,
		HTTPSPort:           443,
		ClientIPStrategy:    "REMOTE-ADDR",
	}
}

/************************
 * Data Accessors
 ************************/

// Schema returns the rosetta schema that describes a Config
func (config Config) Schema() schema.Schema {
	return Schema()
}

// DomainNames returns an array of domains names in this configuration.
func (config Config) DomainNames() []string {

	result := make([]string, len(config.Domains))

	for index := range config.Domains {
		result[index] = config.Domains[index].Hostname
	}

	return result
}

// HTTPPortString returns the HTTP port as a string (prefixed with a colon).
// This defaults to ":80" if no port is specified.
func (config Config) HTTPPortString() (string, bool) {

	if config.HTTPPort == 0 {
		return ":80", false
	}

	return ":" + strconv.Itoa(config.HTTPPort), true
}

// HTTPSPortString returns the HTTPS port as a string (prefixed with a colon).
// This defaults to ":443" if no port is specified.
func (config Config) HTTPSPortString() (string, bool) {

	if config.HTTPSPort == 0 {
		return ":443", false
	}

	return ":" + strconv.Itoa(config.HTTPSPort), true
}

// IsEmpty returns TRUE if this configuration is not usable (no ports or domains)
func (config Config) IsEmpty() bool {

	if config.HTTPPort != 0 {
		return false
	}

	if config.HTTPSPort != 0 {
		return false
	}

	if len(config.Domains) > 0 {
		return false
	}

	return true
}

// IsReadyForDomains returns TRUE if the configuration has the minimum amount of
// data required to add domains to the server.
func (config Config) IsReadyForDomains() bool {

	if config.Templates.IsEmpty() {
		log.Info().Msg("Config: No templates configured")
		return false
	}

	if config.ActivityPubCache.IsEmpty() {
		log.Info().Msg("Config: No ActivityPub cache configured")
		return false
	}

	if config.AttachmentCache.IsEmpty() {
		log.Info().Msg("Config: No attachment cache configured")
		return false
	}

	if config.AttachmentOriginals.IsEmpty() {
		log.Info().Msg("Config: No attachment originals configured")
		return false
	}

	if config.ExportCache.IsEmpty() {
		log.Info().Msg("Config: No export cache configured")
		return false
	}

	if config.Certificates.IsEmpty() {
		log.Info().Msg("Config: No certificates configured")
		return false
	}

	return true
}

// With applies one or more options to this configuration
func (config *Config) With(options ...Option) {

	for _, option := range options {
		option(config)
	}
}
