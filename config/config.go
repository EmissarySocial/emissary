package config

import (
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
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
				"certificates": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"templates": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"layouts": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"static": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"attachmentOriginals": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"attachmentCache": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"adminEmail": schema.String{},
			},
		},
	}
}
