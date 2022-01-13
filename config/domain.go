package config

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
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

func (d *Domain) GetPath(p path.Path) (interface{}, error) {

	switch p.Head() {
	case "label":
		return d.Label, nil
	case "hostname":
		return d.Hostname, nil
	case "connectString":
		return d.ConnectString, nil
	case "databaseName":
		return d.DatabaseName, nil
	case "attachmentPath":
		return d.AttachmentPath, nil
	case "layoutPath":
		return d.LayoutPath, nil
	case "templatePath":
		return d.TemplatePath, nil
	case "forwardTo":
		return d.ForwardTo, nil
	case "showAdmin":
		return d.ShowAdmin, nil

	case "smtp":
		if p.IsTailEmpty() {
			return d.SMTPConnection, nil
		}
		return d.SMTPConnection.GetPath(p.Tail())

	default:
		return nil, derp.New(derp.CodeInternalError, "ghost.config.Domain.GetPath", "Unrecognized config setting", p)
	}
}

func (d *Domain) SetPath(p path.Path, value interface{}) error {

	spew.Dump("domain.SetPath", p, value)
	switch p.Head() {
	case "label":
		d.Label = convert.String(value)
	case "hostname":
		d.Hostname = convert.String(value)
	case "connectString":
		d.ConnectString = convert.String(value)
	case "databaseName":
		d.DatabaseName = convert.String(value)
	case "attachmentPath":
		d.AttachmentPath = convert.String(value)
	case "layoutPath":
		d.LayoutPath = convert.String(value)
	case "templatePath":
		d.TemplatePath = convert.String(value)
	case "forwardTo":
		d.ForwardTo = convert.String(value)
	case "showAdmin":
		d.ShowAdmin = convert.Bool(value)
	case "smtp":
		if p.IsTailEmpty() {
			return derp.New(derp.CodeBadRequestError, "ghost.config.Domain.SetPath", "Cannot set SMTP object directly")
		}

		return d.SMTPConnection.SetPath(p.Tail(), value)

	default:
		return derp.New(derp.CodeInternalError, "ghost.config.Domain.SetPath", "Unrecognized config setting", p)
	}

	return nil
}

func (smtp *SMTPConnection) GetPath(p path.Path) (interface{}, error) {

	if !p.IsTailEmpty() {
		return nil, derp.NewBadRequestError("ghost.config.SMTP.GetPath", "No sub-properties of SMTP connection", p)
	}

	switch p.Head() {
	case "hostname":
		return smtp.Hostname, nil
	case "username":
		return smtp.Username, nil
	case "password":
		return smtp.Password, nil
	case "tls":
		return smtp.TLS, nil
	default:
		return nil, derp.NewBadRequestError("ghost.config.SMTP.GetPath", "Unrecognized path", p)
	}
}

func (smtp *SMTPConnection) SetPath(p path.Path, value interface{}) error {

	spew.Dump("smtp.SetPath", p, value)

	if !p.IsTailEmpty() {
		return derp.NewBadRequestError("ghost.config.SMTP.GetPath", "No sub-properties of SMTP connection", p)
	}

	switch p.Head() {
	case "hostname":
		smtp.Hostname = convert.String(value)
	case "username":
		smtp.Username = convert.String(value)
	case "password":
		smtp.Password = convert.String(value)
	case "tls":
		smtp.TLS = convert.Bool(value)
	default:
		return derp.NewBadRequestError("ghost.config.SMTP.GetPath", "Unrecognized path", p)
	}

	return nil
}
