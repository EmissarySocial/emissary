package service

import (
	"bytes"
	"html/template"
	"io/fs"
	"maps"
	"strings"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"

	mail "github.com/xhit/go-simple-mail/v2"
)

type ServerEmail struct {
	filesystemService Filesystem
	funcMap           template.FuncMap
	emails            map[string]model.Email
}

func NewServerEmail(filesystemService Filesystem, funcMap template.FuncMap, locations []mapof.String) ServerEmail {

	service := ServerEmail{
		filesystemService: filesystemService,
		funcMap:           funcMap,
		emails:            make(map[string]model.Email),
	}

	service.Refresh()

	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *ServerEmail) Refresh() {

	// Reset all emails (to be reloaced by the Template service)
	service.emails = make(map[string]model.Email)
}

/******************************************
 * Real-Time Updates
 ******************************************/

func (service *ServerEmail) Add(filesystem fs.FS, definition []byte) error {

	const location = "service.ServerEmail.Add"

	// Unmarshal the file into the schema.
	temp := mapof.NewAny()
	if err := hjson.Unmarshal(definition, &temp); err != nil {
		return derp.Wrap(err, location, "Error loading Schema")
	}

	email := model.NewEmail(temp.GetString("emailId"), service.funcMap)
	log.Debug().Msg("Email Service: adding " + email.EmailID)

	// Read simple properties
	email.EmailRole = temp.GetString("emailRole")
	email.Model = temp.GetString("model")

	// Read "to"  template
	if toTemplate, err := email.To.Parse(temp.GetString("to")); err == nil {
		email.To = toTemplate
	} else {
		return derp.Wrap(err, location, "Error parsing 'to' template", email.EmailID)
	}

	// Read "subject" template
	if subjectTemplate, err := email.Subject.Parse(temp.GetString("subject")); err == nil {
		email.Subject = subjectTemplate
	} else {
		return derp.Wrap(err, location, "Error parsing 'subject' template", email.EmailID)
	}

	// Read "headers" templates
	for name, value := range temp.GetMap("headers") {
		if headerTemplate, err := email.Headers.New(name).Parse(convert.String(value)); err == nil {
			email.Headers = headerTemplate
		} else {
			return derp.Wrap(err, location, "Error parsing 'headers' template", email.EmailID, name)
		}
	}

	// Read "body" template
	content, err := fs.ReadFile(filesystem, "body.html")

	if err != nil {
		return derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Cannot read body.html file")
	}

	if bodyTemplate, err := email.Body.Parse(string(content)); err == nil {
		email.Body = bodyTemplate
	} else {
		return derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Unable to parse template HTML")
	}

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		email.Resources = resources
	}

	// Add the email into the prep library
	service.emails[email.EmailID] = email

	// Banana
	return nil
}

/******************************************
 * Send Emails API
 ******************************************/

func (service *ServerEmail) Send(smtpConnection config.SMTPConnection, owner config.Owner, emailID string, model string, data mapof.Any) error {

	const location = "service.ServerEmail.Send"

	// Find the email in the library
	email, exists := service.emails[emailID]

	if !exists {
		return derp.BadRequestError(location, "Email is not defined", emailID, maps.Keys(service.emails))
	}

	// "Model" must be set
	if model == "" {
		return derp.BadRequestError(location, "Model is required", emailID)
	}

	// Require that the email is defined for the correct model
	if email.Model != model {
		return derp.BadRequestError(location, "Email is not defined for this model", emailID, model)
	}

	// If the SMTP Connection is empty, then don't try to send an email
	if smtpConnection.IsNil() {
		log.Debug().Str("location", location).Msg("Skipping email because the SMTP Connection is empty.")
		return nil
	}

	// Try to connect to the server
	server, ok := smtpConnection.Server()

	if !ok {
		return derp.InternalError(location, "Cannot create SMTP Connection - invalid or empty credentials", smtpConnection.Hostname, smtpConnection.Username)
	}

	client, err := server.Connect()

	if err != nil {
		return derp.Wrap(err, location, "Error connecting to SMTP server", emailID, data, smtpConnection.Hostname, smtpConnection.Username, strings.Repeat("*", len(smtpConnection.Password)), smtpConnection.Port, smtpConnection.TLS)
	}

	message := mail.NewMSG()
	message.SetFrom(owner.DisplayName + " <" + owner.EmailAddress + ">")

	// Generate the "to" address
	buffer := bytes.Buffer{}
	if err := email.To.Execute(&buffer, data); err != nil {
		return derp.Wrap(err, location, "Error executing 'to' template", emailID, data)
	}
	message.AddTo(buffer.String())
	buffer.Reset()

	// Generate the "subject" line
	if err := email.Subject.Execute(&buffer, data); err != nil {
		return derp.Wrap(err, location, "Error executing 'subject' template", emailID, data)
	}

	message.SetSubject(buffer.String())
	buffer.Reset()

	// Generate the email body
	if err := email.Body.Execute(&buffer, data); err != nil {
		return derp.Wrap(err, location, "Error executing template", emailID, data)
	}

	message.SetBody(mail.TextHTML, buffer.String())
	buffer.Reset()

	// Try to send the email
	if err := message.Send(client); err != nil {
		return derp.Wrap(err, location, "Error sending email", emailID, data)
	}

	return nil
}
