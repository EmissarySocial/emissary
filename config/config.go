package config

import (
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Config defines all of the domains available on this server
type Config struct {
	Domains             set.Slice[string, Domain] `path:"domains"             json:"domains"`             // Slice of one or more domain configurations
	AdminEmail          string                    `path:"adminEmail"          json:"adminEmail"`          // Email address of the administrator
	AttachmentOriginals string                    `path:"attachmentOriginals" json:"attachmentOriginals"` // Folder where original attachments will be stored
	AttachmentCache     string                    `path:"attachmentCache"     json:"attachmentCache"`     // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
	Certificates        string                    `path:"certificates"        json:"certificates"`        // Folder containing the SSL certificate cache for Let's Encrypt AutoSSL
	Layouts             []string                  `path:"layouts"             json:"layouts"`             // Folder containing all system layouts
	Templates           []string                  `path:"templates"           json:"templates"`           // Folder containing all stream templates
	Source              string                    `path:"-"                   json:"-"`                   // Where did the initial config location come from?  (Command Line, Environment Variable, Default)
	Location            string                    `path:"-"                   json:"-"`                   // Location where this config file is read from/to.  Not a part of the configuration itself.
}

// NewConfig returns a fully initialized (but empty) Config data structure.
func NewConfig() Config {

	return Config{
		Domains: make(set.Slice[string, Domain], 0),
	}
}

// DefaultConfig return sthe default configuration for this application.
func DefaultConfig() Config {

	return Config{
		Domains: set.Slice[string, Domain]{{
			DomainID: primitive.NewObjectID().Hex(),
			Label:    "Administration Console",
			Hostname: "localhost",
		}},

		// File Locations
		Layouts:             []string{"embed://layouts"},
		Templates:           []string{"embed://templates"},
		AttachmentOriginals: "file://.emissary/attachments",
		AttachmentCache:     "file://.emissary/cache",
		Certificates:        "file://.emissary/certificates",
	}
}

/************************
 * Data Accessors
 ************************/

// DomainNames returns an array of domains names in this configuration.
func (config Config) DomainNames() []string {

	result := make([]string, len(config.Domains))

	for index := range config.Domains {
		result[index] = config.Domains[index].Hostname
	}

	return result
}

/************************
 * Validating Schema
 ************************/

// Schema returns the data schema for the configuration file.
func Schema() schema.Schema {
	return schema.Schema{
		ID:      "emissary.Server",
		Comment: "Validating schema for a server configuration",
		Element: schema.Object{
			Properties: schema.ElementMap{
				"domains": schema.Array{
					Items: DomainSchema().Element,
				},
				"templates": schema.Array{
					Items:     schema.String{},
					Delimiter: "\n",
				},
				"layouts": schema.Array{
					Items:     schema.String{},
					Delimiter: "\n",
				},
				"attachmentOriginals": schema.String{},
				"attachmentCache":     schema.String{},
				"certificates":        schema.String{},
				"adminEmail":          schema.String{},
			},
		},
	}
}
