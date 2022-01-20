package config

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/list"
	"github.com/benpate/steranko"
)

// Domain contains all of the configuration data required to operate a single domain.
type Domain struct {
	Label          string          `json:"label"`               // Human-friendly label for administrators
	Hostname       string          `json:"hostname"`            // Domain name of a virtual server
	ConnectString  string          `json:"connectString"`       // MongoDB connect string
	DatabaseName   string          `json:"databaseName"`        // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	SMTPConnection SMTPConnection  `json:"smtp"`                // Information for connecting to an SMTP server to send email on behalf of the domain.
	AttachmentPath string          `json:"attachmentPath"`      // Path to attachment directory (TODO: change in the future with afero update)
	LayoutPath     string          `json:"layoutPath"`          // Path to the directory where the website layout is saved.
	TemplatePath   string          `json:"templatePath"`        // Paths to the directory where page templates are defined.
	ForwardTo      string          `json:"forwardTo,omitempty"` // Forwarding information for a domain that has moved servers
	ShowAdmin      bool            `json:"showAdmin"`           // If TRUE, then show domain settings in admin
	Steranko       steranko.Config `json:"steranko,omitempty"`  // Configuration to pass through to Steranko
}

type SMTPConnection struct {
	Hostname string `json:"hostname"` // Server name to connect to
	Username string `json:"username"` // Username for authentication
	Password string `json:"password"` // Password/secret for authentication
	TLS      bool   `json:"tls"`      // If TRUE, then use TLS to connect
}

func NewDomain() Domain {
	return Domain{
		SMTPConnection: SMTPConnection{},
		Steranko:       steranko.Config{},
	}
}

func (d *Domain) GetPath(path string) (interface{}, bool) {

	switch path {
	case "label":
		return d.Label, true
	case "hostname":
		return d.Hostname, true
	case "connectString":
		return d.ConnectString, true
	case "databaseName":
		return d.DatabaseName, true
	case "attachmentPath":
		return d.AttachmentPath, true
	case "layoutPath":
		return d.LayoutPath, true
	case "templatePath":
		return d.TemplatePath, true
	case "forwardTo":
		return d.ForwardTo, true
	case "showAdmin":
		return d.ShowAdmin, true

	}

	head, tail := list.Split(path, ".")

	if head == "smtp" {

		if tail == "" {
			return d.SMTPConnection, true
		}

		return d.SMTPConnection.GetPath(tail)
	}

	return nil, false
}

func (d *Domain) SetPath(path string, value interface{}) error {

	switch path {

	case "label":
		d.Label = convert.String(value)
		return nil

	case "hostname":
		d.Hostname = convert.String(value)
		return nil

	case "connectString":
		d.ConnectString = convert.String(value)
		return nil

	case "databaseName":
		d.DatabaseName = convert.String(value)
		return nil

	case "attachmentPath":
		d.AttachmentPath = convert.String(value)
		return nil

	case "layoutPath":
		d.LayoutPath = convert.String(value)
		return nil

	case "templatePath":
		d.TemplatePath = convert.String(value)
		return nil

	case "forwardTo":
		d.ForwardTo = convert.String(value)
		return nil

	case "showAdmin":
		d.ShowAdmin = convert.Bool(value)
		return nil
	}

	head, tail := list.Split(path, ".")

	if head == "smtp" {

		if tail == "" {
			return derp.New(derp.CodeBadRequestError, "whisper.config.Domain.SetPath", "Cannot set SMTP object directly")
		}

		return d.SMTPConnection.SetPath(tail, value)
	}

	return derp.New(derp.CodeInternalError, "whisper.config.Domain.SetPath", "Unrecognized config setting", path)
}

func (smtp *SMTPConnection) GetPath(path string) (interface{}, bool) {

	if path == "" {
		return smtp, true
	}

	switch path {

	case "hostname":
		return smtp.Hostname, true

	case "username":
		return smtp.Username, true

	case "password":
		return smtp.Password, true

	case "tls":
		return smtp.TLS, true

	default:
		return nil, false
	}
}

func (smtp *SMTPConnection) SetPath(path string, value interface{}) error {

	switch path {

	case "hostname":
		smtp.Hostname = convert.String(value)

	case "username":
		smtp.Username = convert.String(value)

	case "password":
		smtp.Password = convert.String(value)

	case "tls":
		smtp.TLS = convert.Bool(value)

	default:
		return derp.NewBadRequestError("whisper.config.SMTP.GetPath", "Unrecognized path", path)
	}

	return nil
}
