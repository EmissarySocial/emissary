package config

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// Config defines all of the domains available on this server
type Config struct {
	Password    string     `json:"password"`    // Password for access to domain admin console
	Domains     DomainList `json:"domains"`     // Slice of one or more domain configurations
	Templates   Folder     `json:"templates"`   // Folder containing all stream templates
	Layouts     Folder     `json:"layouts"`     // Folder containing all system layouts
	Attachments Folder     `json:"attachments"` // Folder containing all attachments
	Static      Folder     `json:"static"`      // Folder containing all attachments
}

// NewConfig returns a fully initialized (but empty) Config data structure.
func NewConfig() Config {

	return Config{
		Domains: make(DomainList, 0),
	}
}

// DefaultConfig return sthe default configuration for this application.
func DefaultConfig(password string) Config {

	return Config{
		Password: password,
		Domains: DomainList{{
			Label:     "Administration Console",
			Hostname:  "localhost",
			ShowAdmin: true,
		}},
		Attachments: Folder{
			Adapter:  "FILE",
			Location: "./_attachments/",
		},
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
	}
}

/************************
 * Path functions
 ************************/

// GetPath implements the path.Getter interface
func (config Config) GetPath(name string) (interface{}, bool) {

	if name == "password" {
		return config.Password, true
	}

	head, tail := path.Split(name)

	switch head {

	case "layouts":
		return config.Layouts.GetPath(tail)

	case "templates":
		return config.Templates.GetPath(tail)

	case "attachments":
		return config.Attachments.GetPath(tail)

	case "domains":
		return config.Domains.GetPath(tail)
	}

	return nil, false
}

// SetPath implements the path.Setter interface
func (config *Config) SetPath(name string, value interface{}) error {

	if name == "password" {
		config.Password = convert.String(value)
		return nil
	}

	head, tail := path.Split(name)

	switch head {

	case "layouts":
		return config.Layouts.SetPath(tail, value)

	case "templates":
		return config.Templates.SetPath(tail, value)

	case "attachments":
		return config.Attachments.SetPath(tail, value)

	case "domains":
		return config.Domains.SetPath(tail, value)
	}

	return derp.NewInternalError("whisper.config.SetPath", "Unrecognized path", name, value)
}

/************************
 * Other Data Accessors
 ************************/

// DomainNames returns an array of domains names in this configuration.
func (config Config) DomainNames() []string {

	result := make([]string, len(config.Domains))

	for index := range config.Domains {
		result[index] = config.Domains[index].Hostname
	}

	return result
}
