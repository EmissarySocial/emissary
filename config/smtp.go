package config

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
)

type SMTPConnection struct {
	Hostname string `json:"hostname"` // Server name to connect to
	Username string `json:"username"` // Username for authentication
	Password string `json:"password"` // Password/secret for authentication
	TLS      bool   `json:"tls"`      // If TRUE, then use TLS to connect
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
