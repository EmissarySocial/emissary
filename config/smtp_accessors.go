package config

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

func SMTPConnectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"hostname": schema.String{MaxLength: 255},
			"username": schema.String{MaxLength: 255},
			"password": schema.String{MaxLength: 255},
			"port":     schema.Integer{Minimum: null.NewInt64(0), Maximum: null.NewInt64(65535), Required: false},
			"tls":      schema.Boolean{},
		},
	}
}

func (smtp *SMTPConnection) GetPointer(name string) (any, bool) {

	switch name {

	case "hostname":
		return &smtp.Hostname, true

	case "username":
		return &smtp.Username, true

	case "password":
		return &smtp.Password, true

	case "port":
		return &smtp.Port, true

	case "tls":
		return &smtp.TLS, true

	}

	return nil, false
}
