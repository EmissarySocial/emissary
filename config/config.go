package config

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// Config defines all of the domains available on this server
type Config struct {
	Password          string   `json:"password"`          // Password for access to domain admin console
	Domains           []Domain `json:"domains"`           // Slice of one or more domain configurations
	TemplateAdapter   string   `json:"templateAdapter"`   // Type of connection to use to create template adapter
	TemplatePath      string   `json:"templatePath"`      // Path name to use when connecting to template adapter
	AttachmentAdapter string   `json:"attachmentAdapter"` // Type of connection to use to create attachment adapter
	AttachmentPath    string   `json:"attachmentPath"`    // Path name to use when connecting to attachment adapter
}

// NewConfig returns the default configuration for this application.
func NewConfig() Config {
	return Config{
		Domains:           make([]Domain, 0),
		TemplateAdapter:   "File",
		TemplatePath:      "./.templates",
		AttachmentAdapter: "File",
		AttachmentPath:    "./.attachments",
	}
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
 * Path functions
 ************************/

// GetPath implements the path.Getter interface
func (config Config) GetPath(name string) (interface{}, bool) {

	switch name {
	case "password":
		return config.Password, true

	case "templateAdapter":
		return config.TemplateAdapter, true

	case "templatePath":
		return config.TemplatePath, true

	case "attachmentAdapter":
		return config.AttachmentAdapter, true

	case "attachmentPath":
		return config.AttachmentPath, true

	}

	head, tail := path.Split(name)

	if head == "domains" {
		if tail == "" {
			return config.Domains, true
		}

		index, err := path.Index(head, len(config.Domains))

		if err != nil {
			return nil, false
		}

		return path.Get(config.Domains[index], tail), true
	}

	return nil, false
}

// SetPath implements the path.Setter interface
func (config *Config) SetPath(name string, value interface{}) error {

	switch name {
	case "password":
		config.Password = convert.String(value)
		return nil

	case "templateAdapter":
		config.TemplateAdapter = convert.String(value)
		return nil

	case "templatePath":
		config.TemplatePath = convert.String(value)
		return nil

	case "attachmentAdapter":
		config.AttachmentAdapter = convert.String(value)
		return nil

	case "attachmentPath":
		config.AttachmentPath = convert.String(value)
		return nil

	}

	head, tail := path.Split(name)

	if head == "domains" {

		if tail == "" {
			if domains, ok := value.([]Domain); ok {
				config.Domains = domains
				return nil
			}
			return derp.NewInternalError("whisper.config.SetPath", "Cannot set domains", value)
		}

		index, err := path.Index(head, len(config.Domains))

		if err != nil {
			return derp.Wrap(err, "whisper.config.SetPath", "Cannot get slice index", name)
		}

		if err := config.Domains[index].SetPath(tail, value); err != nil {
			return derp.Wrap(err, "whisper.config.SetPath", "Error setting domain")
		}

		return nil
	}

	return derp.NewInternalError("whisper.config.SetPath", "Unrecognized path", name, value)
}
