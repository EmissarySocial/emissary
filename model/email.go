package model

import (
	"io/fs"
	"text/template"
)

type Email struct {
	EmailID   string             // Unique identifier for this email.
	EmailRole string             // Role of the email - for system emails that may have multiple options
	Model     string             // Object type that this email is associated with
	Headers   *template.Template // Additional email header values
	To        *template.Template // Template for the email address to send this email to
	Subject   *template.Template // Template for the email subject
	Body      *template.Template // Template for the email body
	Resources fs.FS              // File system containing additional files (like images) required by this email
}

func NewEmail(emailID string, funcMap template.FuncMap) Email {
	return Email{
		EmailID: emailID,
		Headers: template.New("").Funcs(funcMap),
		To:      template.New("").Funcs(funcMap),
		Subject: template.New("").Funcs(funcMap),
		Body:    template.New("").Funcs(funcMap),
	}
}
