package config

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

func SMTPConnectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"hostname": schema.String{MaxLength: 255, Required: true},
			"username": schema.String{MaxLength: 255, Required: true},
			"password": schema.String{MaxLength: 255, Required: true},
			"port":     schema.Integer{Minimum: null.NewInt64(1), Maximum: null.NewInt64(65535), Required: true},
			"tls":      schema.Boolean{},
		},
	}
}

func (smtp SMTPConnection) GetBoolOK(name string) (bool, bool) {

	switch name {

	case "tls":
		return smtp.TLS, true
	}

	return false, false
}

func (smtp SMTPConnection) GetIntOK(name string) (int, bool) {

	switch name {

	case "port":
		return smtp.Port, true
	}

	return 0, false
}

func (smtp SMTPConnection) GetStringOK(name string) (string, bool) {

	switch name {

	case "hostname":
		return smtp.Hostname, true

	case "username":
		return smtp.Username, true

	case "password":
		return smtp.Password, true
	}

	return "", false
}

func (smtp *SMTPConnection) SetBool(name string, value bool) bool {

	switch name {

	case "tls":
		smtp.TLS = value
		return true
	}

	return false
}

func (smtp *SMTPConnection) SetInt(name string, value int) bool {

	switch name {

	case "port":
		smtp.Port = value
		return true

	}

	return false
}

func (smtp *SMTPConnection) SetString(name string, value string) bool {

	switch name {

	case "hostname":
		smtp.Hostname = value
		return true

	case "username":
		smtp.Username = value
		return true

	case "password":
		smtp.Password = value
		return true

	}

	return false
}
