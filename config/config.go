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
	Layouts             []Folder                  `path:"layouts"             json:"layouts"`             // Folders containing all system layouts
	Templates           []Folder                  `path:"templates"           json:"templates"`           // Folders containing all stream templates
	Emails              []Folder                  `path:"emails"              json:"emails"`              // Folders containing email templates
	AttachmentOriginals Folder                    `path:"attachmentOriginals" json:"attachmentOriginals"` // Folder where original attachments will be stored
	AttachmentCache     Folder                    `path:"attachmentCache"     json:"attachmentCache"`     // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
	Certificates        Folder                    `path:"certificates"        json:"certificates"`        // Folder containing the SSL certificate cache for Let's Encrypt AutoSSL
	AdminEmail          string                    `path:"adminEmail"          json:"adminEmail"`          // Email address of the administrator
	Source              string                    `path:"-"                   json:"-"`                   // READONLY: Where did the initial config location come from?  (Command Line, Environment Variable, Default)
	Location            string                    `path:"-"                   json:"-"`                   // READONLY: Location where this config file is read from/to.  Not a part of the configuration itself.
	MongoID             primitive.ObjectID        `path:"configId"            json:"_" bson:"_id"`        // Used as unique key for MongoDB
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
		Domains: set.Slice[string, Domain]{},

		// File Locations
		Layouts:             []Folder{{Adapter: "EMBED", Location: "layouts"}},
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

/************************
 * Validating Schema
 ************************/

// Schema returns the data schema for the configuration file.
func Schema() schema.Schema {

	readableFileElement := schema.Object{
		Properties: schema.ElementMap{
			"adapter":  schema.String{Required: true, Default: "EMBED", Enum: []string{"EMBED", "FILE", "GIT", "HTTP", "S3"}},
			"location": schema.String{Required: true, MaxLength: null.NewInt(1000)},
		},
	}

	writableFileElement := schema.Object{
		Properties: schema.ElementMap{
			"adapter":  schema.String{Required: true, Default: "FILE", Enum: []string{"FILE", "S3"}},
			"location": schema.String{Required: true, MaxLength: null.NewInt(1000)},
		},
	}

	return schema.Schema{
		ID:      "emissary.Server",
		Comment: "Validating schema for a server configuration",
		Element: schema.Object{
			Properties: schema.ElementMap{
				"domains":             schema.Array{Items: DomainSchema().Element},
				"templates":           schema.Array{Items: readableFileElement, MinLength: null.NewInt(1)},
				"layouts":             schema.Array{Items: readableFileElement, MinLength: null.NewInt(1)},
				"emails":              schema.Array{Items: readableFileElement, MinLength: null.NewInt(1)},
				"attachmentOriginals": writableFileElement,
				"attachmentCache":     writableFileElement,
				"certificates":        writableFileElement,
				"adminEmail":          schema.String{Format: "email"},
			},
		},
	}
}
