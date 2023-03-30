package service

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"

	mail "github.com/xhit/go-simple-mail/v2"
)

type ServerEmail struct {
	filesystemService Filesystem
	funcMap           template.FuncMap
	locations         []mapof.String
	templates         *template.Template

	changed chan bool
	closed  chan bool
}

func NewServerEmail(filesystemService Filesystem, funcMap template.FuncMap, locations []mapof.String) ServerEmail {

	service := ServerEmail{
		filesystemService: filesystemService,
		funcMap:           funcMap,
		changed:           make(chan bool),
		closed:            make(chan bool),
	}

	service.Refresh(locations)

	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *ServerEmail) Refresh(locations sliceof.Object[mapof.String]) {

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
	if err := service.loadTemplates(); err != nil {
		derp.Report(derp.Wrap(err, "service.Template.Refresh", "Error loading templates from filesystem"))
		return
	}

	// Try to watch the template directory for changes
	go service.watch()
}

/******************************************
 * REAL-TIME UPDATES
 ******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *ServerEmail) watch() {

	// abort the existing watcher
	close(service.closed)

	// open a new channel for the next watcher
	service.closed = make(chan bool)

	// Start new watchers.
	for _, folder := range service.locations {

		if err := service.filesystemService.Watch(folder, service.changed, service.closed); err != nil {
			derp.Report(derp.Wrap(err, "service.Layout.Watch", "Error watching filesystem", folder))
		}
	}

	// All Watchers Started.  Now Listen for Changes
	for {

		select {

		case <-service.changed:
			service.loadTemplates()

		case <-service.closed:
			return
		}
	}
}

func (service *ServerEmail) loadTemplates() error {

	templates := template.New("")

	for _, location := range service.locations {

		filesystem, err := service.filesystemService.GetFS(location)

		if err != nil {
			return derp.Report(err)
		}

		if err := loadHTMLTemplateFromFilesystem(filesystem, templates, service.funcMap); err != nil {
			return derp.Report(err)
		}

		fmt.Println("... email: " + location["location"])
	}

	// If we got this far, then we're good to go!
	service.templates = templates

	return nil
}

func (service *ServerEmail) Send(smtpConnection config.SMTPConnection, templateName string, from string, to []string, subject string, data any) error {

	const location = "service.ServerEmail.Send"

	// Build the email body
	var buffer bytes.Buffer
	if err := service.templates.ExecuteTemplate(&buffer, templateName, data); err != nil {
		return derp.Wrap(err, location, "Error executing template", templateName, data)
	}

	// Build the email message
	message := mail.NewMSG()
	message.SetFrom(from).
		AddTo(to...).
		SetSubject(subject).
		SetBody(mail.TextHTML, buffer.String())

	// Try to connect to the server
	server, ok := smtpConnection.Server()

	if !ok {
		return derp.NewInternalError(location, "ServerEmail service is not configured")
	}

	client, err := server.Connect()

	if err != nil {
		return derp.Wrap(err, location, "Error connecting to SMTP server", templateName, from, to, subject, data)
	}

	// Try to send the email
	if err := message.Send(client); err != nil {
		return derp.Wrap(err, location, "Error sending email", templateName, from, to, subject, data)
	}

	return nil
}
