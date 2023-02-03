package config

import (
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Config defines all of the domains available on this server
type Config struct {
	Domains             set.Slice[Domain]      `json:"domains"`             // Slice of one or more domain configurations
	Providers           set.Slice[Provider]    `json:"providers"`           // Slice of one or more OAuth client configurations
	Themes              sliceof.Object[Folder] `json:"themes"`              // Folders containing all system themes
	Templates           sliceof.Object[Folder] `json:"templates"`           // Folders containing all stream templates
	Emails              sliceof.Object[Folder] `json:"emails"`              // Folders containing email templates
	AttachmentOriginals Folder                 `json:"attachmentOriginals"` // Folder where original attachments will be stored
	AttachmentCache     Folder                 `json:"attachmentCache"`     // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
	Certificates        Folder                 `json:"certificates"`        // Folder containing the SSL certificate cache for Let's Encrypt AutoSSL
	AdminEmail          string                 `json:"adminEmail"`          // Email address of the administrator
	Source              string                 `json:"-"`                   // READONLY: Where did the initial config location come from?  (Command Line, Environment Variable, Default)
	Location            string                 `json:"-"`                   // READONLY: Location where this config file is read from/to.  Not a part of the configuration itself.
	MongoID             primitive.ObjectID     `json:"_" bson:"_id"`        // Used as unique key for MongoDB
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
		Themes:              []Folder{{Adapter: "EMBED", Location: "themes"}},
		Templates:           []Folder{{Adapter: "EMBED", Location: "templates"}},
		Emails:              []Folder{{Adapter: "EMBED", Location: "emails"}},
		AttachmentOriginals: Folder{Adapter: "FILE", Location: ".emissary/attachments"},
		AttachmentCache:     Folder{Adapter: "FILE", Location: ".emissary/cache"},
		Certificates:        Folder{Adapter: "FILE", Location: ".emissary/certificates"},
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
