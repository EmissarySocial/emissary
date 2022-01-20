package config

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/benpate/steranko"
)

// Domain contains all of the configuration data required to operate a single domain.
type Domain struct {
	Label          string          `json:"label"`               // Human-friendly label for administrators
	Hostname       string          `json:"hostname"`            // Domain name of a virtual server
	ConnectString  string          `json:"connectString"`       // MongoDB connect string
	DatabaseName   string          `json:"databaseName"`        // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	SMTPConnection SMTPConnection  `json:"smtp"`                // Information for connecting to an SMTP server to send email on behalf of the domain.
	ForwardTo      string          `json:"forwardTo,omitempty"` // Forwarding information for a domain that has moved servers
	ShowAdmin      bool            `json:"showAdmin"`           // If TRUE, then show domain settings in admin
	Steranko       steranko.Config `json:"steranko,omitempty"`  // Configuration to pass through to Steranko
}

func NewDomain() Domain {
	return Domain{
		SMTPConnection: SMTPConnection{},
		Steranko:       steranko.Config{},
	}
}

func (d *Domain) GetPath(name string) (interface{}, bool) {

	switch name {
	case "label":
		return d.Label, true

	case "hostname":
		return d.Hostname, true

	case "connectString":
		return d.ConnectString, true

	case "databaseName":
		return d.DatabaseName, true

	case "forwardTo":
		return d.ForwardTo, true

	case "showAdmin":
		return d.ShowAdmin, true

	}

	head, tail := path.Split(name)

	if head == "smtp" {

		if tail == "" {
			return d.SMTPConnection, true
		}

		return d.SMTPConnection.GetPath(tail)
	}

	return nil, false
}

func (d *Domain) SetPath(name string, value interface{}) error {

	switch name {

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

	case "forwardTo":
		d.ForwardTo = convert.String(value)
		return nil

	case "showAdmin":
		d.ShowAdmin = convert.Bool(value)
		return nil
	}

	head, tail := path.Split(name)

	if head == "smtp" {

		if tail == "" {
			return derp.New(derp.CodeBadRequestError, "whisper.config.Domain.SetPath", "Cannot set SMTP object directly")
		}

		return d.SMTPConnection.SetPath(tail, value)
	}

	return derp.New(derp.CodeInternalError, "whisper.config.Domain.SetPath", "Unrecognized config setting", name)
}
