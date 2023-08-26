/*
Package config includes definitions for the Emissary configuration file, along with
adapters for reading/writing from the filesystem or a mongodb database.
*/
package config

import (
	"strconv"

	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Config defines all of the domains available on this server
type Config struct {
	Domains             set.Slice[Domain]            `json:"domains"`             // Slice of one or more domain configurations
	Providers           set.Slice[Provider]          `json:"providers"`           // Slice of one or more OAuth client configurations
	Templates           sliceof.Object[mapof.String] `json:"templates"`           // Folders containing all stream templates
	Emails              sliceof.Object[mapof.String] `json:"emails"`              // Folders containing email templates
	AttachmentOriginals mapof.String                 `json:"attachmentOriginals"` // Folder where original attachments will be stored
	AttachmentCache     mapof.String                 `json:"attachmentCache"`     // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
	Certificates        mapof.String                 `json:"certificates"`        // Folder containing the SSL certificate cache for Let's Encrypt AutoSSL
	ActivityPubCache    mapof.String                 `json:"activityPubCache"`    // Connection string for ActivityPub cache database
	AdminEmail          string                       `json:"adminEmail"`          // Email address of the administrator
	HTTPPort            int                          `json:"httpPort"`            // Port to listen on for HTTP requests
	HTTPSPort           int                          `json:"httpsPort"`           // Port to listen on for HTTPS requests
	DebugLevel          string                       `json:"debugLevel"`          // Amount of debugging information to log for the server (None, Terse, Verbose)
	Source              string                       `json:"-"`                   // READONLY: Where did the initial config location come from?  (Command Line, Environment Variable, Default)
	Location            string                       `json:"-"`                   // READONLY: Location where this config file is read from/to.  Not a part of the configuration itself.
	MongoID             primitive.ObjectID           `json:"-" bson:"_id"`        // Used as unique key for MongoDB
}

// NewConfig returns a fully initialized (but empty) Config data structure.
func NewConfig() Config {
	return Config{
		Domains: make(set.Slice[Domain], 0),
	}
}

// DefaultConfig return sthe default configuration for this application.
func DefaultConfig() Config {

	return Config{
		Domains: set.Slice[Domain]{},

		// File Locations
		Templates:           sliceof.Object[mapof.String]{mapof.String{"adapter": "EMBED", "location": "templates"}},
		Emails:              sliceof.Object[mapof.String]{mapof.String{"adapter": "EMBED", "location": "emails"}},
		AttachmentOriginals: mapof.String{"adapter": "FILE", "location": ".emissary/attachments"},
		AttachmentCache:     mapof.String{"adapter": "FILE", "location": ".emissary/cache"},
		Certificates:        mapof.String{"adapter": "FILE", "location": ".emissary/certificates"},
		ActivityPubCache:    mapof.String{},
		DebugLevel:          "None",
		HTTPPort:            80,
		HTTPSPort:           443,
	}
}

/************************
 * Data Accessors
 ************************/

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

func (config Config) AllProviders() []form.LookupCode {

	// Just locate the providers that require configuration
	allProviders := slice.Filter(dataset.Providers(), func(lookupCode form.LookupCode) bool {
		return (lookupCode.Group != "MANUAL")
	})

	// Use the Group field to show if the provider is active or not.
	allProviders = slice.Map(allProviders, func(lookupCode form.LookupCode) form.LookupCode {
		provider, _ := config.Providers.Get(lookupCode.Value)
		if provider.IsEmpty() {
			lookupCode.Group = ""
		} else {
			lookupCode.Group = "ACTIVE"
		}
		return lookupCode
	})

	return allProviders
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
