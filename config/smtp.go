package config

import (
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	mail "github.com/xhit/go-simple-mail/v2"
)

type SMTPConnection struct {
	Hostname string `json:"hostname"` // Server name to connect to
	Username string `json:"username"` // Username for authentication
	Password string `json:"password"` // Password/secret for authentication
	Port     int    `json:"port"`     // Port to connect to
	TLS      bool   `json:"tls"`      // If TRUE, then use TLS to connect
}

func NewSMTPConnection() SMTPConnection {
	return SMTPConnection{}
}

// IsNil returns TRUE if the SMTPConnection is not populated with any information
func (smtp SMTPConnection) IsNil() bool {
	return smtp.Hostname == ""
}

// Validate confirms that the SMTPConnection matches ths SMTPConnectionSchema
func (smtp SMTPConnection) Validate() error {
	// RULE: pass a POINTER. The schema resolves properties through GetPointer,
	// which is declared on a pointer receiver, so a value copy does not satisfy
	// the schema's PointerGetter interface and every property fails validation.
	_, err := schema.New(SMTPConnectionSchema()).Validate(&smtp)
	return err
}

// Server generates a fully initialized SMTP server object.
// This object may still be invalid, if the SMTPConnection is not populated with correct information.
func (smtp SMTPConnection) Server() (*mail.SMTPServer, bool) {

	if smtp.Validate() != nil {
		return nil, false
	}

	result := mail.NewSMTPClient()

	result.Host = smtp.Hostname
	result.Port = smtp.Port
	result.Username = smtp.Username
	result.Password = smtp.Password

	if smtp.TLS {
		result.Encryption = mail.EncryptionSSLTLS
	} else {
		result.Encryption = mail.EncryptionNone
	}

	return result, true
}

// TestConnection verifies that the mail server is actually reachable and that the credentials work,
// then disconnects WITHOUT sending a message.  It mirrors the database TestConnection so a typo'd
// hostname, wrong port, bad password, or mismatched TLS setting fails loudly at configuration time
// instead of silently, months later, the first time a member needs a password reset.  `timeout`
// bounds the connect so an unreachable host fails in seconds rather than hanging the setup request.
func (smtp SMTPConnection) TestConnection(timeout time.Duration) error {

	const location = "config.SMTPConnection.TestConnection"

	// RULE: An empty SMTP block means "no email configured" -- a legitimate, optional state.
	// There is nothing to reach, so the test passes.
	if smtp.IsNil() {
		return nil
	}

	// Build the server.  This also runs Validate(), so a malformed block (e.g. a port out of
	// range) is reported before we ever open a socket.
	server, ok := smtp.Server()

	if !ok {
		return derp.Validation("Mail server settings are incomplete or invalid", smtp.Hostname, smtp.Port)
	}

	// Bound the connect so an unreachable host fails in seconds, not on the library's default.
	// KeepAlive is off because this is a throwaway connection we close immediately.
	server.ConnectTimeout = timeout
	server.SendTimeout = timeout
	server.KeepAlive = false

	// Connect opens the TCP socket, negotiates TLS, and authenticates -- exercising hostname, port,
	// TLS, and credentials together -- without sending a message.
	client, err := server.Connect()

	if err != nil {
		return derp.Wrap(err, location, "Unable to reach the mail server. Please check the hostname, port, credentials, and TLS setting.", smtp.Hostname, smtp.Port, smtp.TLS)
	}

	// Politely end the SMTP session and release the socket.  A failure here does not matter --
	// the connection already succeeded, which is all we were testing.
	_ = client.Quit()

	return nil
}
