package service

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/textproto"
	"regexp"
	"slices"
	"strings"
	texttemplate "text/template"
	"text/template/parse"

	"github.com/benpate/rosetta/maps"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"

	mail "github.com/xhit/go-simple-mail/v2"
)

// ServerEmail holds every email template on this server, and sends messages built from them
type ServerEmail struct {
	filesystemService Filesystem
	funcMap           template.FuncMap
	emails            map[string]model.Email
}

// NewServerEmail returns a fully initialized ServerEmail service, loaded from the provided locations
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

// Refresh updates this service with the latest configuration values
func (service *ServerEmail) Refresh() {

	// Reset all emails (to be reloaced by the Template service)
	service.emails = make(map[string]model.Email)
}

/******************************************
 * Real-Time Updates
 ******************************************/

// Add parses an email definition and registers it in this service's library
func (service *ServerEmail) Add(filesystem fs.FS, definition []byte) error {

	const location = "service.ServerEmail.Add"

	// Unmarshal the file into the schema.
	temp := mapof.NewAny()
	if err := hjson.Unmarshal(definition, &temp); err != nil {
		return derp.Wrap(err, location, "Loading Schema")
	}

	// RULE: an email definition must identify itself, because the ID is how Send() finds it
	emailID := temp.GetString("emailId")

	if emailID == "" {
		return derp.BadRequest(location, "Email definition must include an 'emailId'")
	}

	email := model.NewEmail(emailID, service.funcMap)
	log.Debug().Msg("Email Service: adding " + email.EmailID)

	// Read simple properties
	email.EmailRole = temp.GetString("emailRole")
	email.Model = temp.GetString("model")

	// RULE: every definition declares the object its data describes, which is what RequireModel()
	// compares a Go sender's fixed data shape against.  This is deliberately not checked against
	// templateModelRegistry: an email names the object the message is ABOUT (such as "Follower"),
	// which is a different namespace from a Template's builder model.
	if email.Model == "" {
		return derp.BadRequest(location, "Email definition must include a 'model'", email.EmailID)
	}

	// RULE: a definition with no recipient renders an empty address, which fails at send time
	// -- once per attempted delivery -- rather than here, once, at startup
	to := temp.GetString("to")

	if to == "" {
		return derp.BadRequest(location, "Email definition must include a 'to' address", email.EmailID)
	}

	// Read "to"  template
	toTemplate, err := email.To.Parse(to)

	if err != nil {
		return derp.Wrap(err, location, "Parsing 'to' template", email.EmailID)
	}

	email.To = toTemplate

	// Read "subject" template
	subjectTemplate, err := email.Subject.Parse(temp.GetString("subject"))

	if err != nil {
		return derp.Wrap(err, location, "Parsing 'subject' template", email.EmailID)
	}

	email.Subject = subjectTemplate

	// Read "headers" templates
	for name, value := range temp.GetMap("headers") {

		// RULE: header names are written into the message verbatim, without the encoding that
		// protects values, so a name carrying CRLF or a colon would forge headers of its own
		if !headerNamePattern.MatchString(name) {
			return derp.BadRequest(location, "Invalid email header name", email.EmailID, name)
		}

		// RULE: reserved headers are owned by Send() or by the mail library.  A definition that
		// set one could redirect the message to another recipient, disguise who it is from, or
		// corrupt the MIME structure that carries the body.
		if slices.Contains(reservedHeaderNames, textproto.CanonicalMIMEHeaderKey(name)) {
			return derp.BadRequest(location, "Email header name is reserved", email.EmailID, name)
		}

		headerTemplate, err := email.Headers.New(name).Parse(convert.String(value))

		if err != nil {
			return derp.Wrap(err, location, "Parsing 'headers' template", email.EmailID, name)
		}

		email.Headers = headerTemplate
	}

	// Read "body" template
	content, err := fs.ReadFile(filesystem, "body.html")

	if err != nil {
		return derp.Wrap(err, location, "Cannot read body.html file", email.EmailID)
	}

	bodyTemplate, err := email.Body.Parse(string(content))

	if err != nil {
		return derp.Wrap(err, location, "Parsing 'body' template", email.EmailID)
	}

	email.Body = bodyTemplate

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		email.Resources = resources
	}

	// RULE: a later filesystem location may deliberately override an email that an earlier one
	// defined, so a duplicate is legal -- but an accidental collision is otherwise silent
	if _, exists := service.emails[email.EmailID]; exists {
		log.Warn().Str("emailId", email.EmailID).Msg("Email Service: replacing a previously-defined email")
	}

	// Add the email into the prep library
	service.emails[email.EmailID] = email

	// Banana
	return nil
}

// Names returns the ID of every email template in this service's library, sorted
func (service *ServerEmail) Names() []string {
	result := maps.Keys(service.emails)
	slices.Sort(result)
	return result
}

/******************************************
 * Load-Time Queries
 ******************************************/

// Exists returns TRUE if an email with this ID is defined in this service's library
func (service *ServerEmail) Exists(emailID string) bool {
	_, exists := service.emails[emailID]
	return exists
}

// RequiredKeys returns every data key that an email's "to" and "headers" templates interpolate.
// These are the templates that reject a missing key outright, so a caller that omits one of these
// does not render a blank value -- it fails the whole send.  Keys that Send supplies for every
// email are excluded, because no caller has to pass them.
func (service *ServerEmail) RequiredKeys(emailID string) sliceof.String {

	email, exists := service.emails[emailID]

	if !exists {
		return sliceof.String{}
	}

	found := make(mapof.Bool)
	collectTreeFieldNames(email.To, found)

	for _, headerTemplate := range email.Headers.Templates() {
		collectTreeFieldNames(headerTemplate, found)
	}

	result := make(sliceof.String, 0, len(found))

	for name := range found {
		if !slices.Contains(providedKeys, name) {
			result = append(result, name)
		}
	}

	slices.Sort(result)
	return result
}

// RequireModel returns an error unless the named email is defined for modelName.  Callers that
// build a fixed data shape in Go use this to reject a definition -- possibly one an administrator
// overrode on disk -- that describes some other object entirely.
func (service *ServerEmail) RequireModel(emailID string, modelName string) error {

	const location = "service.ServerEmail.RequireModel"

	email, exists := service.emails[emailID]

	if !exists {
		return derp.BadRequest(location, "Email is not defined", emailID, maps.Keys(service.emails))
	}

	if modelName == "" {
		return derp.BadRequest(location, "Model is required", emailID)
	}

	if email.Model != modelName {
		return derp.BadRequest(location, "Email requires a different model object", "email: "+emailID, "required model: "+email.Model, "requested model: "+modelName)
	}

	// A model citizen
	return nil
}

/******************************************
 * Send Emails API
 ******************************************/

// Send renders the named email template and delivers it over the provided SMTP connection
func (service *ServerEmail) Send(smtpConnection config.SMTPConnection, owner config.Owner, emailID string, data mapof.Any) error {

	const location = "service.ServerEmail.Send"

	// RULE: `data` never appears in an error report below -- it holds interpolation values
	// that include password reset codes and, for web-form templates, whatever a visitor typed

	// Find the email in the library
	email, exists := service.emails[emailID]

	if !exists {
		return derp.BadRequest(location, "Email is not defined", emailID, maps.Keys(service.emails))
	}

	// If the SMTP Connection is empty, then don't try to send an email
	if smtpConnection.IsNil() {
		log.Debug().Str("location", location).Msg("Skipping email because the SMTP Connection is empty.")
		return nil
	}

	// Try to connect to the server
	server, ok := smtpConnection.Server()

	if !ok {
		return derp.Internal(location, "Cannot create SMTP Connection - invalid or empty credentials", smtpConnection.Hostname, smtpConnection.Username)
	}

	client, err := server.Connect()

	if err != nil {
		return derp.Wrap(err, location, "Connecting to SMTP server", emailID, smtpConnection.Hostname, smtpConnection.Username, strings.Repeat("*", len(smtpConnection.Password)), smtpConnection.Port, smtpConnection.TLS)
	}

	// RULE: go-simple-mail closes this connection only from inside message.Send(), so every
	// error return before that point would leak it.  A double close is harmless
	defer func() {
		_ = client.Close()
	}()

	message := mail.NewMSG()
	message.SetFrom(owner.DisplayName + " <" + owner.EmailAddress + ">")

	// Generate the "to" address
	buffer := bytes.Buffer{}
	if err := email.To.Execute(&buffer, data); err != nil {
		return derp.Wrap(err, location, "Executing 'to' template", emailID)
	}
	message.AddTo(buffer.String())
	buffer.Reset()

	// Generate the "subject" line
	if err := email.Subject.Execute(&buffer, data); err != nil {
		return derp.Wrap(err, location, "Executing 'subject' template", emailID)
	}

	message.SetSubject(buffer.String())
	buffer.Reset()

	// Generate the custom headers (such as Reply-To) that this Email defines
	if err := applyHeaders(message, email, data); err != nil {
		return derp.Wrap(err, location, "Applying 'headers' templates", emailID)
	}

	// RULE: go-simple-mail keeps its first error and no-ops every later setter, so a bad
	// address is otherwise only reported by Send(), without naming the header that failed
	if err := message.GetError(); err != nil {
		return derp.Wrap(err, location, "Invalid email header", emailID)
	}

	// Generate the email body
	if err := email.Body.Execute(&buffer, data); err != nil {
		return derp.Wrap(err, location, "Executing 'body' template", emailID)
	}

	message.SetBody(mail.TextHTML, buffer.String())
	buffer.Reset()

	// Try to send the email
	if err := message.Send(client); err != nil {
		return derp.Wrap(err, location, "Sending email", emailID)
	}

	// You've got mail
	return nil
}

/******************************************
 * Header Utilities
 ******************************************/

// headerNamePattern matches an RFC 5322 field name: printable US-ASCII,
// excluding the colon that terminates the name (%d33-57 and %d59-126)
var headerNamePattern = regexp.MustCompile(`^[!-9;-~]+$`)

// reservedHeaderNames are the headers an email definition may not set: three that decide who
// receives the message, three that decide who it claims to be from, and four owned by the mail
// library.  Reply-To is deliberately absent -- setting it is the reason "headers" exists.
// Compared in canonical MIME form, since go-simple-mail canonicalizes before it stores them.
var reservedHeaderNames = []string{
	"To", "Cc", "Bcc",
	"From", "Sender", "Return-Path",
	"Date", "Mime-Version", "Content-Type", "Content-Transfer-Encoding",
}

// applyHeaders renders every custom header that an Email defines, and adds it to the message
func applyHeaders(message *mail.Email, email model.Email, data mapof.Any) error {

	const location = "service.applyHeaders"

	// An Email that declares no headers has nothing to apply
	if email.Headers == nil {
		return nil
	}

	// Collect the header names, skipping the set's unnamed root template, which holds no value
	templates := email.Headers.Templates()
	names := make([]string, 0, len(templates))

	for _, headerTemplate := range templates {
		if name := headerTemplate.Name(); name != "" {
			names = append(names, name)
		}
	}

	// Sort the names so that a message's headers are emitted in a stable order
	slices.Sort(names)

	// Render each header in turn
	buffer := bytes.Buffer{}

	for _, name := range names {

		buffer.Reset()

		if err := email.Headers.ExecuteTemplate(&buffer, name, data); err != nil {
			return derp.Wrap(err, location, "Executing 'headers' template", email.EmailID, name)
		}

		// Headers that render empty are skipped: go-simple-mail parses address headers with
		// mail.ParseAddress, where an empty string is an error that fails the whole send
		value := strings.TrimSpace(buffer.String())

		if value == "" {
			continue
		}

		message.AddHeader(name, value)
	}

	// Return to sender.  Address unknown
	return nil
}

// providedKeys are supplied by DomainEmail.Send for every email, so no caller passes them
var providedKeys = []string{"Domain_Owner", "Domain_URL", "Domain_Name", "Domain_Icon"}

// collectTreeFieldNames walks one template's parse tree, skipping templates that never parsed
func collectTreeFieldNames(tmpl *texttemplate.Template, result mapof.Bool) {

	if (tmpl == nil) || (tmpl.Tree == nil) {
		return
	}

	collectFieldNames(tmpl.Tree.Root, result)
}

// collectFieldNames walks a parsed template and records the top-level field name of every
// value it interpolates, so that "{{.ReplyEmail}}" and "{{if .Unsubscribe}}" both yield one name
func collectFieldNames(node parse.Node, result mapof.Bool) {

	switch typed := node.(type) {

	case *parse.ListNode:
		if typed != nil {
			for _, child := range typed.Nodes {
				collectFieldNames(child, result)
			}
		}

	case *parse.ActionNode:
		collectFieldNames(typed.Pipe, result)

	case *parse.PipeNode:
		if typed != nil {
			for _, command := range typed.Cmds {
				collectFieldNames(command, result)
			}
		}

	case *parse.CommandNode:
		for _, argument := range typed.Args {
			collectFieldNames(argument, result)
		}

	case *parse.IfNode:
		collectFieldNames(typed.Pipe, result)
		collectFieldNames(typed.List, result)
		collectFieldNames(typed.ElseList, result)

	case *parse.RangeNode:
		collectFieldNames(typed.Pipe, result)
		collectFieldNames(typed.List, result)
		collectFieldNames(typed.ElseList, result)

	case *parse.WithNode:
		collectFieldNames(typed.Pipe, result)
		collectFieldNames(typed.List, result)
		collectFieldNames(typed.ElseList, result)

	case *parse.FieldNode:
		// Only the FIRST identifier is a data key: "{{.Actor.Name}}" requires "Actor"
		if len(typed.Ident) > 0 {
			result[typed.Ident[0]] = true
		}
	}
}
