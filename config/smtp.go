package config

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	mail "github.com/xhit/go-simple-mail/v2"
)

type SMTPConnection struct {
	Hostname string `path:"hostname" json:"hostname"` // Server name to connect to
	Username string `path:"username" json:"username"` // Username for authentication
	Password string `path:"password" json:"password"` // Password/secret for authentication
	Port     int    `path:"port"     json:"port"`     // Port to connect to
	TLS      bool   `path:"tls"      json:"tls"`      // If TRUE, then use TLS to connect
}

func SMTPConnectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"hostname": schema.String{MaxLength: null.NewInt(255), Required: true},
			"username": schema.String{MaxLength: null.NewInt(255), Required: true},
			"password": schema.String{MaxLength: null.NewInt(255), Required: true},
			"port":     schema.Integer{Minimum: null.NewInt64(1), Maximum: null.NewInt64(65535), Required: true},
			"tls":      schema.Boolean{},
		},
	}
}

// Validate confirms that the SMTPConnection matches ths SMTPConnectionSchema
func (smtp SMTPConnection) Validate() error {
	schema := SMTPConnectionSchema()
	result := schema.Validate(smtp)
	derp.Report(result)
	return result
}

// Server generates a fully initialized SMTP server object.
// This object may still be invalid, if the SMTPConnection is not populated with correct information.
func (smtp *SMTPConnection) Server() (*mail.SMTPServer, bool) {

	if smtp.Validate() != nil {
		return nil, false
	}

	result := mail.NewSMTPClient()

	result.Host = smtp.Hostname
	result.Port = smtp.Port
	result.Username = smtp.Username
	result.Password = smtp.Password

	return result, true
}
