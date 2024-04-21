package service

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog/log"

	mail "github.com/xhit/go-simple-mail/v2"
)

type ServerEmail struct {
	filesystemService Filesystem
	funcMap           template.FuncMap
	locations         []mapof.String
	templates         *template.Template

	refresh chan channel.Done
}

func NewServerEmail(filesystemService Filesystem, funcMap template.FuncMap, locations []mapof.String) ServerEmail {

	service := ServerEmail{
		filesystemService: filesystemService,
		funcMap:           funcMap,
		refresh:           make(chan channel.Done),
	}

	service.Refresh(locations)

	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *ServerEmail) Refresh(locations sliceof.Object[mapof.String]) {

	// Reset the "refresh" channel
	close(service.refresh)
	service.refresh = make(chan channel.Done)

	// RULE: If the Filesystem is empty, then don't try to load
	if len(locations) == 0 {
		return
	}

	// RULE: If nothing has changed since the last time we refreshed, then we're done.
	if slicesAreEqual(locations, service.locations) {
		return
	}

	// Add configuration to the service
	service.locations = locations

	// Load all templates from the filesystem
	service.loadTemplates()

	// Try to watch the template directory for changes
	go service.watch()
}

/******************************************
 * REAL-TIME UPDATES
 ******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *ServerEmail) watch() {

	changes := make(chan bool)
	defer close(changes)

	// Start new watchers.
	for _, folder := range service.locations {

		if err := service.filesystemService.Watch(folder, changes, service.refresh); err != nil {
			derp.Report(derp.Wrap(err, "service.Layout.Watch", "Error watching filesystem", folder))
		}
	}

	// All Watchers Started.  Now Listen for Changes
	for {
		select {
		case <-changes:
			service.loadTemplates()
		case <-service.refresh:
			return
		}

	}
}

func (service *ServerEmail) loadTemplates() {

	templates := template.New("")

	for _, location := range service.locations {

		log.Trace().Msg("Server Email Service: adding email: " + location["location"])

		filesystem, err := service.filesystemService.GetFS(location)

		if err != nil {
			derp.Report(err)
		}

		if err := loadHTMLTemplateFromFilesystem(filesystem, templates, service.funcMap); err != nil {
			derp.Report(err)
		}

	}

	// If we got this far, then we're good to go!
	service.templates = templates
}

func (service *ServerEmail) Send(smtpConnection config.SMTPConnection, message *mail.Email, templateName string, data any) error {

	const location = "service.ServerEmail.Send"

	spew.Dump("serverEmail.Send", data)

	// If the SMTP Connection is empty, then don't try to send an email
	if smtpConnection.IsNil() {
		log.Debug().Msg("ServerEmail.Send: SMTP Connection is empty.  Skipping email.")
		return nil
	}

	log.Trace().Str("template", templateName).Msg("ServerEmail.Send: sending email")

	// Build the email body
	var buffer bytes.Buffer
	if err := service.templates.ExecuteTemplate(&buffer, templateName, data); err != nil {
		return derp.Wrap(err, location, "Error executing template", templateName, data)
	}

	// Build the email message
	message.SetBody(mail.TextHTML, buffer.String())

	// Try to connect to the server
	server, ok := smtpConnection.Server()

	if !ok {
		return derp.NewInternalError(location, "Cannot create SMTP Connection - invalid or empty credentials", smtpConnection.Hostname, smtpConnection.Username)
	}

	client, err := server.Connect()

	if err != nil {
		return derp.Wrap(err, location, "Error connecting to SMTP server", templateName, data, smtpConnection.Hostname, smtpConnection.Username, strings.Repeat("*", len(smtpConnection.Password)), smtpConnection.Port, smtpConnection.TLS)
	}

	// Try to send the email
	if err := message.Send(client); err != nil {
		return derp.Wrap(err, location, "Error sending email", templateName, data)
	}

	return nil
}
