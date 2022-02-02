package config

// Config defines all of the domains available on this server
type Config struct {
	AdminURL            string     `path:"adminUrl"            json:"adminUrl"`            // path to use for the server admin console (if blank, then console is not available)
	AdminPassword       string     `path:"adminPassword"       json:"adminPassword"`       // Password for access to domain admin console
	Domains             DomainList `path:"domains"             json:"domains"`             // Slice of one or more domain configurations
	Templates           Folder     `path:"templates"           json:"templates"`           // Folder containing all stream templates
	Layouts             Folder     `path:"layouts"             json:"layouts"`             // Folder containing all system layouts
	Static              Folder     `path:"static"              json:"static"`              // Folder containing all attachments
	AttachmentOriginals Folder     `path:"attachmentOriginals" json:"attachmentOriginals"` // Folder where original attachments will be stored
	AttachmentCache     Folder     `path:"attachmentCache"     json:"attachmentCache"`     // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
}

// NewConfig returns a fully initialized (but empty) Config data structure.
func NewConfig() Config {

	return Config{
		Domains: make(DomainList, 0),
	}
}

// DefaultConfig return sthe default configuration for this application.
func DefaultConfig(adminURL string, adminPassword string) Config {

	// If Admin URL is empty, then blank out the password, too
	if adminURL == "" {
		adminPassword = ""
	} else {
		// Otherwise, add a prefix to be clear that there's no overlap with Stream URLs
		adminURL = "." + adminURL
	}

	return Config{
		AdminURL:      adminURL,
		AdminPassword: adminPassword,
		Domains: DomainList{{
			Label:     "Administration Console",
			Hostname:  "localhost",
			ShowAdmin: true,
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
