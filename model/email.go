package model

import (
	htmltemplate "html/template"
	"io/fs"
	texttemplate "text/template"
)

// Email is a single email template, along with the resources it needs to render
type Email struct {
	EmailID   string                 // Unique identifier for this email.
	EmailRole string                 // Role of the email - for system emails that may have multiple options
	Model     string                 // Object type that this email is associated with
	Headers   *texttemplate.Template // Additional email header values (plain-text context)
	To        *texttemplate.Template // Template for the email address to send this email to (plain-text context)
	Subject   *texttemplate.Template // Template for the email subject (plain-text context)
	Body      *htmltemplate.Template // Template for the HTML email body. This uses html/template so that
	// interpolated values are contextually auto-escaped, exactly like web pages. Using text/template here
	// would let user- or remote-controlled data inject arbitrary HTML into outgoing emails (CWE-79/CWE-116).
	Resources fs.FS // File system containing additional files (like images) required by this email
}

// NewEmail creates an empty Email. funcMap is an html/template.FuncMap (the same map the web
// templates use); text/template accepts the same map type, so it is shared across all four templates.
func NewEmail(emailID string, funcMap htmltemplate.FuncMap) Email {
	return Email{
		EmailID: emailID,
		Headers: texttemplate.New("").Funcs(texttemplate.FuncMap(funcMap)),
		To:      texttemplate.New("").Funcs(texttemplate.FuncMap(funcMap)),
		Subject: texttemplate.New("").Funcs(texttemplate.FuncMap(funcMap)),
		Body:    htmltemplate.New("").Funcs(funcMap),
	}
}
