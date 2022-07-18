package config

import (
	"github.com/EmissarySocial/emissary/tools/set"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Config defines all of the domains available on this server
type Config struct {
	Domains             set.Slice[string, Domain] `path:"domains"             json:"domains"`             // Slice of one or more domain configurations
	Certificates        Folder                    `path:"certificates"        json:"certificates"`        // Folder containing the SSL certificate cache for Let's Encrypt AutoSSL
	Templates           Folder                    `path:"templates"           json:"templates"`           // Folder containing all stream templates
	Layouts             Folder                    `path:"layouts"             json:"layouts"`             // Folder containing all system layouts
	Static              Folder                    `path:"static"              json:"static"`              // Folder containing all attachments
	AttachmentOriginals Folder                    `path:"attachmentOriginals" json:"attachmentOriginals"` // Folder where original attachments will be stored
	AttachmentCache     Folder                    `path:"attachmentCache"     json:"attachmentCache"`     // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
	AdminEmail          string                    `path:"adminEmail"          json:"adminEmail"`          // Email address of the administrator
	Bootstrap           string                    `path:"-"                   json:"-"`                   // Location to look for the config file
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
		Layouts: Folder{
			Adapter:  "FILE",
			Location: "./_layouts/",
			Sync:     true,
		},
		Static: Folder{
			Adapter:  "FILE",
			Location: "./_static/",
		},
		Templates: Folder{
			Adapter:  "FILE",
			Location: "./_templates/",
			Sync:     true,
		},
		Certificates: Folder{
			Adapter:  "FILE",
			Location: "./_certificates/",
			Sync:     false,
		},
		AttachmentOriginals: Folder{
			Adapter:  "FILE",
			Location: "./_attachments/originals",
			Sync:     false,
		},
		AttachmentCache: Folder{
			Adapter:  "FILE",
			Location: "./_attachments/cache",
			Sync:     false,
		},
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
